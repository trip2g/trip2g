package jsonneteval

import (
	"errors"
	"reflect"
	"time"
	"unsafe"

	jsonnet "github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
)

const (
	// DefaultMaxDepth bounds nested node evaluations. go-jsonnet's MaxStack
	// skips tailstrict calls and super chains, and each nesting level costs
	// Go stack; a fatal Go stack overflow cannot be recovered.
	DefaultMaxDepth = 20000
	// DefaultMaxAdditions bounds evaluated + operators. Each one can add a
	// layer to an object or string that go-jsonnet later walks recursively
	// outside node evaluation, where MaxDepth does not see it.
	DefaultMaxAdditions = 100000
	DefaultTimeout      = 5 * time.Second

	deadlineCheckEvery = 1024
)

var (
	ErrTooDeep          = errors.New("jsonnet evaluation is nested too deeply")
	ErrTooManyAdditions = errors.New("jsonnet evaluation uses + too many times")
	ErrTimeout          = errors.New("jsonnet evaluation took too long")
)

// VM is a jsonnet VM whose every evaluation is bounded in depth, additions
// and time.
type VM struct {
	*jsonnet.VM
	MaxDepth     int
	MaxAdditions int
	Timeout      time.Duration
	budget       *budget
}

type budget struct {
	depth        int
	maxDepth     int
	additions    int
	maxAdditions int
	steps        int
	deadline     time.Time
	err          error
}

// EvaluateAnonymousSnippet shadows jsonnet.VM's method so no caller can run
// a snippet without the budget.
func (vm *VM) EvaluateAnonymousSnippet(name, src string) (string, error) {
	*vm.budget = budget{
		maxDepth:     vm.MaxDepth,
		maxAdditions: vm.MaxAdditions,
		deadline:     time.Now().Add(vm.Timeout),
	}

	out, err := vm.VM.EvaluateAnonymousSnippet(name, src)
	if vm.budget.err != nil {
		return "", vm.budget.err
	}

	return out, err
}

func (b *budget) pre(_ unsafe.Pointer, node ast.Node) {
	b.depth++
	if b.depth > b.maxDepth {
		b.abort(ErrTooDeep)
	}

	if bin, ok := node.(*ast.Binary); ok && bin.Op == ast.BopPlus {
		b.additions++
		if b.additions > b.maxAdditions {
			b.abort(ErrTooManyAdditions)
		}
	}

	b.steps++
	if b.steps%deadlineCheckEvery == 0 && time.Now().After(b.deadline) {
		b.abort(ErrTimeout)
	}
}

func (b *budget) post(_ unsafe.Pointer, _ ast.Node, _ any, _ error) {
	b.depth--
}

// abort unwinds the evaluation; go-jsonnet recovers the panic at the top of
// EvaluateAnonymousSnippet, and the caller reports b.err instead.
func (b *budget) abort(err error) {
	b.err = err
	panic(err)
}

// installBudget sets go-jsonnet's EvalHook, which it calls around every node
// evaluation but only exports to its own debugger. Its funcs take the
// unexported *interpreter and value, which are passed here as an opaque
// pointer and an interface of the same machine layout. checkEvalHookLayout
// panics instead if a go-jsonnet upgrade changes that shape.
func installBudget(vm *jsonnet.VM, b *budget) {
	hook := reflect.ValueOf(&vm.EvalHook).Elem()
	checkEvalHookLayout(hook)

	*(*func(unsafe.Pointer, ast.Node))(unsafe.Pointer(hook.FieldByName("pre").UnsafeAddr())) = b.pre
	*(*func(unsafe.Pointer, ast.Node, any, error))(unsafe.Pointer(hook.FieldByName("post").UnsafeAddr())) = b.post
}

func checkEvalHookLayout(hook reflect.Value) {
	nodeType := reflect.TypeFor[ast.Node]()
	errorType := reflect.TypeFor[error]()

	pre, preOK := hook.Type().FieldByName("pre")
	post, postOK := hook.Type().FieldByName("post")
	if !preOK || !postOK {
		panic("jsonneteval: go-jsonnet EvalHook has no pre/post hooks")
	}

	ok := pre.Type.Kind() == reflect.Func && pre.Type.NumIn() == 2 && pre.Type.NumOut() == 0 &&
		pre.Type.In(0).Kind() == reflect.Pointer && pre.Type.In(1) == nodeType &&
		post.Type.Kind() == reflect.Func && post.Type.NumIn() == 4 && post.Type.NumOut() == 0 &&
		post.Type.In(0).Kind() == reflect.Pointer && post.Type.In(1) == nodeType &&
		post.Type.In(2).Kind() == reflect.Interface && post.Type.In(3) == errorType
	if !ok {
		panic("jsonneteval: go-jsonnet EvalHook changed shape: " + pre.Type.String() + ", " + post.Type.String())
	}
}
