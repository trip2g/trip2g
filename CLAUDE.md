Read instructions of common patterns in docs/instructions.md

## Golang

Don’t write

```golang
if err := ...; err != nil
```

Always use two lines:

```golang
err = ...
if err != nil {
```

**IMPORTANT**: After making changes to Go code:
1. Run tests for affected packages: `go test ./internal/case/packagename -v`
2. Run all tests to ensure nothing is broken: `go test ./...`
3. Run `make lint` to ensure:
   - Code compiles without errors
   - All linting rules pass
   - Generated code is up to date

## Commit Guidelines

When creating commits, follow these guidelines:

### Message Format
- Use conventional commit format: `type(scope): description`
- Keep first line under 50 characters when possible
- Use present tense: "add feature" not "added feature"
- Use imperative mood: "fix bug" not "fixes bug"

### Common Types
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code refactoring without functionality change
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

### Examples
```
feat(ui/admin): add release management catalog
fix(db): handle null values in user queries
refactor(ui): move components to proper namespaces
docs: update API documentation
```

### Commit Process
```bash
git add .
git commit -m "type(scope): brief description"
```

Do not add co-author comments or generated signatures unless specifically requested.

## Technology Stack

**Backend:**
- Go 1.21+ with SQLite (WAL mode)
- [sqlc](https://sqlc.dev/) for type-safe SQL queries
- [gqlgen](https://gqlgen.com/) for GraphQL server
- [fasthttp](https://github.com/valyala/fasthttp) for HTTP server
- [ozzo-validation](https://github.com/go-ozzo/ozzo-validation) for input validation
- [dbmate](https://github.com/amacneil/dbmate) for database migrations

**Frontend:**
- [$mol framework](https://github.com/hyoo-ru/mam_mol) with TypeScript

## Development Workflow

### Backend Changes
1. **SQL**: Add queries to `queries.sql` → run `make sqlc`
2. **GraphQL**: Update `internal/graph/schema.graphqls` → run `make gqlgen`
3. **Business Logic**: Implement in `internal/case/.../resolve.go`
4. **Tests**: Write comprehensive tests with table-driven patterns

### Frontend Changes
1. **Components**: Create `.view.tree` files for structure
2. **Behavior**: Add `.view.ts` files for TypeScript behavior
3. **GraphQL**: Use `$trip2g_graphql_request` → run `npm run graphqlgen`
4. **Organization**: Group by entity (e.g., `admin/noteview/select/`)

## Key Patterns

### GraphQL Mutations
- Accept only one `input` argument
- Return `union ${Mutation}OrErrorPayload = ${Mutation}Payload | ErrorPayload`
- Use Env interface pattern for testability

### SQL Style Guide
- **Keywords**: Use lowercase for all SQL keywords (`select`, `from`, `where`, `create table`, etc.)
- **Table/Column names**: Use lowercase with underscores
- **Indentation**: Use consistent indentation for readability
- **Example**:
  ```sql
  -- Good
  create table users (
      id integer primary key,
      email text not null,
      created_at datetime not null default (datetime('now'))
  );
  
  -- Bad
  CREATE TABLE Users (
      ID INTEGER PRIMARY KEY,
      Email TEXT NOT NULL,
      CreatedAt DATETIME NOT NULL DEFAULT (DATETIME('now'))
  );
  ```

## Adding New Features

### Adding SQL Queries and Database Methods

When you need new database operations:

1. **Add SQL Query to `queries.sql`**:
   ```sql
   -- name: MethodName :one
   select * from table_name where id = ?;
   ```

2. **Generate Go Code**:
   ```bash
   make sqlc
   ```

3. **Check Generated Method** in `internal/db/queries.sql.go`:
   ```go
   func (q *Queries) MethodName(ctx context.Context, id int64) (TableType, error)
   ```

4. **Add to Env Interface** (if needed for GraphQL resolvers):
   - The main `Env` interface automatically includes all `*Queries` methods
   - For case-specific interfaces, add method to the case's `Env` interface

### Adding GraphQL Mutations

1. **Check Schema** in `internal/graph/schema.graphqls`:
   - Mutation may already be defined
   - Input/Output types should follow pattern: `${Mutation}Input`, `${Mutation}Payload`, `${Mutation}OrErrorPayload`

2. **Run GraphQL Generation**:
   ```bash
   make gqlgen
   ```

3. **Implement Business Logic**:
   - Create directory: `internal/case/${mutationname}/` (for user mutations) or `internal/case/admin/${mutationname}/` (for admin mutations)
   - Create `resolve.go` following this pattern:
     ```go
     package ${mutationname}

     import (
         "context"
         "database/sql"
         "fmt"

         ozzo "github.com/go-ozzo/ozzo-validation/v4"
         validation "github.com/go-ozzo/ozzo-validation/v4"
         "github.com/go-ozzo/ozzo-validation/v4/is"

         "trip2g/internal/db"
         "trip2g/internal/graph/model"
     )

     type Env interface {
         // Required database methods
         InsertSomething(ctx context.Context, arg db.InsertSomethingParams) error
         // Other methods as needed
     }

     // Type aliases for cleaner code
     type Input = model.${Mutation}Input
     type Payload = model.${Mutation}OrErrorPayload

     // validateRequest validates input and returns ErrorPayload if invalid
     func validateRequest(r *Input) *model.ErrorPayload {
         return model.NewOzzoError(ozzo.ValidateStruct(r,
             ozzo.Field(&r.Field1, validation.Required),
             ozzo.Field(&r.Email, validation.Required, is.Email),
             // Add all validation rules
         ))
     }

     func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
         // Always validate input first
         errPayload := validateRequest(&input)
         if errPayload != nil {
             return errPayload, nil  // User-visible errors go in ErrorPayload
         }

         // Define params as separate variable for cleaner code
         params := db.InsertSomethingParams{
             Field1: input.Field1,
             Field2: sql.NullString{String: input.Field2, Valid: input.Field2 != ""},
             // Map all fields
         }

         // Execute database operation
         err := env.InsertSomething(ctx, params)
         if err != nil {
             // System errors are returned as error (will show generic message to user)
             return nil, fmt.Errorf("failed to insert something: %w", err)
         }

         // Define payload as separate variable
         payload := model.${Mutation}Payload{
             Success: true,
             // Add other return fields
         }

         return &payload, nil
     }
     ```

   **Important patterns:**
   - Use type aliases (`Input`, `Payload`) for cleaner code
   - Create `validateRequest` function that returns `*model.ErrorPayload`
   - User-visible validation errors return `ErrorPayload` with `nil` error
   - System/internal errors return `nil` payload with wrapped error
   - Define params and payload as separate variables before use
   - Return `&payload, nil` for successful responses

4. **Define Env Interface** in the case:
   ```go
   type Env interface {
       RequiredMethod1(ctx context.Context, ...) (Type, error)
       RequiredMethod2(ctx context.Context, ...) error
   }
   ```

5. **Add Case Env to Main Interface** in `internal/graph/resolver.go`:
   ```go
   import "trip2g/internal/case/${mutationname}"        // for user mutations
   import "trip2g/internal/case/admin/${mutationname}"  // for admin mutations
   
   type Env interface {
       // ... existing methods ...
       ${mutationname}.Env
   }
   ```

6. **Update GraphQL Resolver** in `internal/graph/schema.resolvers.go`:
   ```go
   // For user mutations (in root Mutation type):
   import "trip2g/internal/case/${mutationname}"
   
   func (r *mutationResolver) ${Mutation}(ctx context.Context, input model.${Mutation}Input) (model.${Mutation}OrErrorPayload, error) {
       return ${mutationname}.Resolve(ctx, r.env(ctx), input)
   }
   
   // For admin mutations (in AdminMutation type):
   import "trip2g/internal/case/admin/${mutationname}"
   
   func (r *adminMutationResolver) ${Mutation}(ctx context.Context, obj *appmodel.AdminMutation, input model.${Mutation}Input) (model.${Mutation}OrErrorPayload, error) {
       return ${mutationname}.Resolve(ctx, r.env(ctx), input)
   }
   ```

7. **Write Tests** following the pattern in `internal/userbans/userbans_test.go`:
   - Create `resolve_test.go` with table-driven tests
   - Use `//go:generate moq` for mocking
   - Test success, error, and edge cases
   - **Don't forget**: Run `go generate` if tests contain `//go:generate moq` to generate mocks

8. **Add Methods to Main Server** (if needed) in `cmd/server/main.go`:
   - Only if the case requires methods not available in standard `*Queries`

### Frontend Components
- **Naming**: `$trip2g_admin_entity_action` format
- **Lists**: Use `$trip2g_graphql_make_map()` with `row(id)` methods

### Testing
- **Libraries**: `github.com/kr/pretty`, `github.com/matryer/moq`, `github.com/stretchr/testify/require`
- **Pattern**: Table-driven tests with mock setup functions
- **Mocks**: Generate with `//go:generate moq -out mocks_test.go . Env`
- **Error handling**: Always use two-line pattern in tests

**IMPORTANT - Refactoring Policy:**
- **Before refactoring**: Always ensure the code has comprehensive tests
- **If no tests exist**: Propose writing tests first before any refactoring
- **Test coverage**: Tests should cover main functionality, error cases, and edge cases
- **Refactoring safety**: Never refactor code without proper test coverage

## Mol framework

1. **View Definitions**
   - Tree-based UI specs live under `ui/` in `.view.tree` files, with behavior in corresponding `.view.ts`.
   - List views use `$trip2g_graphql_request` + `$trip2g_graphql_make_map()`, then define `row(id)` and column getters. Always add id for each row for `$trip2g_graphql_make_map`.

2. **Routing & Linking**
   - The main `ui/admin/admin.view.tree` uses `spreads` to switch pages by `nav` arg.
   - List trees wire detail pages via `Content* $trip2g_admin_show_X` and `param \x_id <= row_id*`.

3. **Detail/Edit Pages**
   - Show pages fetch a single record via GraphQL query in their `.view.ts`.
   - Use `$mol_labeler`, `$mol_date`, `$mol_time_moment`, etc., for form controls.
   - Bind inputs two-way using `<=>`, e.g., `value_moment? <=> expires_at_moment?`.

4. GraphQL Requests
    - Use `$trip2g_graphql_request` for GraphQL queries and mutations.
      Run `npm run graphqlgen` to regenerate TypeScript types after modifying GraphQL schema.

### $mol_view (about view.tree)

The base class for all visual components. It provides the infrastructure for reactive lazy rendering, handling exceptions. By default it finds or creates a `div` without child node changing and additional attributes, fields and event handler creation. You can customize it by inheritance or properties overriding at instantiating.

## Properties

**`dom_name()' : string`**

Returns name of the DOM-element creating for component, if the element with appropriate id is not presented at DOM yet.

**`dom_name_space() = 'http://www.w3.org/1999/xhtml'`**

Returns namespaceURI for the DOM element. 

**`sub() : Array< $mol_view | Node | string | number | boolean > = null `**

Returns list of child components/elements/primitives. If the list have not been set (by default), then the content of the DOM-element would not be changed in way, it's helpful for manual operating with DOM.

**`context( next? : $ ) : $`**
Some rendering context. Parent node injects context to all rendered child components.

**`minimal_height()` = 0**

Returns minimum possible height of the component. It's set by hand with constant or some expression.This property is used for lazy rendering.

**`dom_node() : Element`**

Returns DOM-element, to which the component is bounded to. At first the method tries to find the element by its id at DOM and only if it would have not been found - the method would create and remember a new one. 

**`dom_tree() : Element`**

Same as `dom_node`, but its guarantee, that the content, attributes and properties of the DOM-element should be in actual state.

**`attr() : { [ key : string ] : string | number | boolean }`**

Returns the dictionary of the DOM-attributes, which values would be set while rendering. Passing `null` or `false` as the value to the attribute would lead to removing the attribute.
Passing `true` is an equivalent to passing its name as value. `undefined` is just ignored.

**`field() : { [ key : string ] : any }`**

Returns dictionary of fields, which is necessary to set to the DOM-element after rendering.

**`style() : { [ key : string ] : string | number }`**

Returns dictionary of styles. Numbers will be converted to string with "px" suffix.

**`event() : { [ key : string ] : ( event : Event )=> void }`**

Returns dictionary of event handlers. The event handlers are bind to the DOM-element one time, when the value is set to `dom_node` property. This handlers are synchronous and can be cancelled by ```preventDefault()```.

**`focused( next? : boolean ) : boolean`**

Determines, whether the component is focused or not at this time. If any inserted component would be focused, then its parent component would be focused also.

**`plugins() : Array< $mol_view > = null`**

Array of plugins. Plugin is a component which can be supplemented with the logic of the current components.

For example, list component with keyboard navigation (used `$mol_nav` plugin):

```
<= Options $mol_list
    plugins / 
        <= Nav $mol_nav
            keys_y <= options /
    rows <= options /
```

## *.view.tree

*view.tree* - is a declarative language of describing components, based on [tree format](https://github.com/nin-jin/tree.d). One file can have multiple component definitions, but better to put every component in a separate file, except in very trivial cases.
To create a new component in `view.tree` file you must inherit it from any existing one or `$mol_view`.
Name of the component should begin with `$` and be unique globally accordance with principles presented on [MAM](https://github.com/eigenmethod/mam). For example, let's declare the component `$my_button` extended from `$mol_view`:

```tree
$my_button $mol_view
```

It translates to (every *.view.tree code would be translated to *.view.tree.ts):

```typescript
namespace $ { export class $my_button extends $mol_view {} }
```

When inheriting, it is possible to declare additional properties or overload existing ones (but the property type must match). For example, lets overload a `uri` property with `"https://example.org"` string, and `sub` - with array of one string `"Click me!"`, besides, lets declare a new property `target` with `"_top"` value by default (default value is necessary when declaring a new property):

```tree
$my_example $mol_link
	uri \https://example.org
	sub /
		\Click me!
	target \_top
```

```typescript
namespace $ { export class $my_example extends $mol_link {

	uri() { return "https://example.org" }

	sub() { return [ "Click me!" ] }

	target() { return "_top" }
} }
```

Note: For better readability, a single child node in a tree file is often written on a single line. You can expand all nodes in the previous example:

```tree
$my_example
	$mol_link
		uri
			\https://example.org
		sub
			/
				\Click me!
		target
			\_top
```

Node where name starts with `$` - name of component.
Child nodes beginning with node `/` - list. You can set type of list, e.g `/number`, `/$mol_view` for better type checking.
Text after `\` - raw data which can contain entirely any data until the end of the line.
Node `@` marks string for extraction to separate `*.locale=en.json` file and used for i18n translation, e.g `@ \Values example`.
Numbers, booleans values and `null` is being wrote as it is, without any prefixes:
Nodes after `-` are ignored, you can use them for commenting and temporary disable subtree.

```tree
$my_values $mol_view
	title @ \Values example
	sub /
		0
		1.1
		true
		false
		null
		\I can contain any character! \("o")/
		- I
			am
				remark...
```

```typescript
namespace $ { export class $my_values extends $mol_view {

	title() {
		return this.$.$mol_locale.text( '$my_values_title' )
	}

	sub() {
		return [ 0 , 1.1 , true , false , <any> null , "I can contain any character! \\(\"o\")/" ]
	}

} }
````

Dictionary (correspondence keys to their values) could be declared through a node `*` (you can use `^` to inherit pairs from superclass). For example, set DOM-element's attribute values:

```tree
$my_number $mol_view
	dom_name \input
	attr *
		^
		type \number
		- attribute values must be a strings
		min \0
		max \20
```

```typescript
namespace $ { export class $my_number extends $mol_view {

	dom_name() { return "input" }

	attr() {
		return { ...super.attr() ,
			"type" : "number" ,
			"min" : "0" ,
			"max" : "20" ,
		}
	}
} }
```

To set a value for a DOM element's fields:

```tree
$my_scroll $mol_view
	field *
		^
		scrollTop 0
```

```typescript
namespace $ { export class $my_scroll extends $mol_view {

	field() {
		return { ...super.field() ,
			"scrollTop" : 0 ,
		}
	}
} }
```

To set styles:

```tree
$my_rotate $mol_view
	style *
		^
		transform \rotate( 180deg )
```

```typescript
namespace $ { export class $my_rotate extends $mol_view {

	style() {
		return { ...super.style() ,
			"transform" : "rotate( 180deg )" ,
		}
	}
} }
```

As a value, we could cast not only constants, but also the contents of other properties through `<=` one-way binding. For example, let's declare two text properties `hint` and `text` and then use them for the `field` dictionary and `sub` list:

```tree
$my_hint $mol_view
	hint \Default hint
	text \Default text
	field *
		^
		title <= hint -
	sub /
		<= text -
```

```typescript
namespace $ { export class $my_hint extends $mol_view {

	hint() { return "Default hint" }

	text() { return "Default text" }

	field() {
		return { ...super.field() ,
			"title" : this.hint() ,
		}
	}

	sub() {
		return [ this.text() ]
	}
} }
```

It's often convenient to combine declaring a property and using it. The following example is exactly the same as the previous one:

```tree
$my_hint $mol_view
	field *
		^
		title <= hint \Default hint 
	sub /
		<= text \Default text
```

Reactions on DOM-events are required for two-way binding. For example, lets point out, that objects of `click` event is necessary to put in `remove` property, which we declare right here and set it a default value `null`:

```tree
$my_remover $mol_view
	event *
		^
		click? <=> remove? null 
	sub /
		\Remove
```

```typescript
namespace $ { export class $my_remover extends $mol_view {

	@ $mol_mem
	remove( next? : any ) {
		return ( next !== undefinded ) ? next : null as any
	}

	event() {
		return { ...super.event() ,
			"click" : ( next? : any )=> this.remove( next ) ,
		}
	}

	sub() {
		return [ "Remove" ]
	}
} }
```

You can declare an instance of another class as a value directly. The following example declares a `List` property with the value of instance `$mol_list_demo_tree` and then places it in a list of `sub` child components:

```tree
$my_app $mol_view
	List $mol_list_demo_tree
	sub /
		<= List -
```

```typescript
namespace $ { export class $my_app extends $mol_view {

	@ $mol_mem
	List() {
		const obj = new $mol_list_demo_tree
		return obj
	}

	sub() {
		return [ this.List() ]
	}

} }
```

Properties of a nested component can be overloaded, below overloaded `title` and `content` properties of `$mol_label` component: 

```tree
$my_name $mol_view
	sub /
		<= Info $mol_label
			title \Name
			content \Jin
```

```typescript
namespace $ { export class $my_name extends $mol_view {

	@ $mol_mem
	Info() {
		const obj = new $mol_label
		obj.title = () => "Name"
		obj.content = () => "Jin"
		return obj
	}

	sub() {
		return [ this.Info() ]
	}
} }
```

Properties of parent and child components can be linked. In the following example, we declare a reactive `name` property and tell the `Input` child component to use the `name` property as its own `value` property, we also tell the `Output` child component that we want the `name` property to output inside that.
The `Input` and `Output` components are linked through the `name` parent property, and changing the value in the `Input` will also update the `Output`:

```tree
$my_greeter $mol_view
	sub /
		<= Input $mol_string
			hint \Name
			value? <=> name? \
		<= Output $mol_view
			sub /
				<= name? \
```

```typescript
namespace $ { export class $my_greeter extends $mol_view {

	@ $mol_mem
	name( next? : any ) {
		return ( next !== undefined ) ? next : ""
	}

	@ $mol_mem
	Input() {
		const obj = new $mol_string
		obj.hint = () => "Name"
		obj.value = ( next? : any ) => this.name( next )
		return obj
	}

	@ $mol_mem
	Output() {
		const obj = new $mol_view
		obj.sub = () => [ this.name() ]
		return obj
	}

	sub() {
		return [ this.Input() , this.Output() ]
	}

} }
```

`=>` - Right-side binding. It declares alias for property of subcomponent in declared component.

```
$my_app $mol_scroll
	sub /
		<= Page $mol_page
			Title => Page_title -
			head /
				<= Back $mol_button_minor
					title \Back
				<= Page_title -
```

```typescript
namespace $ {
	export class $my_app extends $mol_scroll {
		
		// sub / <= Page
		sub() {
			return [
				this.Page()
			] as readonly any[]
		}
		
		// Back $mol_button_minor title \Back
		@ $mol_mem
		Back() {
			const obj = new this.$.$mol_button_minor()
			
			obj.title = () => "Back"
			
			return obj
		}
		
		// Page_title
		Page_title() {
			return this.Page().Title()
		}
		
		// Page $mol_page
		// 	Title => Page_title
		// 	head /
		// 		<= Back
		// 		<= Page_title
		@ $mol_mem
		Page() {
			const obj = new this.$.$mol_page()
			
			obj.head = () => [
				this.Back(),
				this.Page_title()
			] as readonly any[]
			
			return obj
		}
	}
}
```

There are certain properties that return different values depending on the key. A typical example of is a list of strings. Each row is a separate component, accessed by a unique key. The list of such properties has a `*` after the name:

```tree
$my_tasks $mol_list
	sub <= task_rows /
	Task_row* $mol_view
		sub /
			<= task_title* <= task_title_default \
```

```typescript
namespace $ {
	export class $my_tasks extends $mol_list {
		
		// sub <= task_rows
		sub() { return this.task_rows() }
		
		// Task_row* $mol_view sub / <= task_title*
		@ $mol_mem_key
		Task_row(id: any) {
			const obj = new this.$.$mol_view()
			
			obj.sub = () => [
				this.task_title(id)
			] as readonly any[]
			
			return obj
		}
		
		// task_rows /
		task_rows() { return [] as readonly any[] }
		
		// task_title_default \
		task_title_default() { return "" }
		
		// task_title* <= task_title_default
		task_title(id: any) { return this.task_title_default() }
	}	
}
```

In above example we declared the property `Task_row`, which takes on input some ID-key and returns an unique instance of `$mol_view` for every key. 
`Task_row` has overloaded property `sub` that outputs appropriate `task_title` for every `Task_row` (`task_title` returns content of property `task_title_default`), which is equal to empty string initially.
Further, by overloading any of these properties, we can change any aspect of the component's behavior. You can override `task_rows` in a subclass to generate rows of your choice. For example":

```
task_rows() {
	const rows = [] as $mol_view[]
	for( let i = 0 ; i < 10 ; ++ i ) rows.push( this.Task_row( i ) )
	return rows
}

task_title(id: any) {
   return `Title - ${id}`
}
```

Note: There is old way howto to use property with id. Instead of `*` you can write `!id`. E.g. instead of `task_title*` you can use `task_title!key`. You might find such usage in old examples, tutorials or old projects.


### All special chars

- `-` - remarks, ignored by code generation
- `$` - component name prefix, e.g `$mol_button`
- `/` - array, optionally you can set type of array, e.g `sub /number`
- `*` - dictionary (string keys, any values)
- `^` - return value of the same property from super class
- `\` - raw string, e.g. `message \Hello`
- `@` - localized string, e.g. `message @ \Hello world`
- `<=` - provides read-only property from owner to sub-componen
- `=>` - provides read-only property from sub-componen to owner
- `<=>` - fully replace sub component property by owner's one
- property + `*` or `!` - property takes ID as first argument, e.g. `Task_row* $mol_view`
- property + `?` - property can be changed by providing an additional optional argument, e.g. `value <=> name? \`

## Date/Time Formatting

### $mol_time_moment

When working with dates and times in frontend components, use `$mol_time_moment` for proper formatting:

**Date Input Controls:**
- Use `$mol_date` in `.view.tree` files for date inputs
- Bind with `value_moment? <=> date_moment? null`

**Time/DateTime Formatting for GraphQL:**
- Use `moment.toString()` to format for Go backend (produces ISO8601 format)
- **For date-only inputs** (from `$mol_date`), convert to full datetime:
  ```typescript
  // Convert date-only to full datetime format for Go
  startsAt: this.starts_at_moment() ? new $mol_time_moment(this.starts_at_moment().toString() + 'T00:00:00Z').toString() : null,
  endsAt: this.ends_at_moment() ? new $mol_time_moment(this.ends_at_moment().toString() + 'T23:59:59Z').toString() : null,
  ```
- **For datetime inputs** (from `$mol_time_moment` with time):
  ```typescript
  createdAt: this.created_at_moment()?.toString() || null,
  ```

**Patterns:**
- `toString()` produces `YYYY-MM-DDThh:mm:ss.sssZ` (default ISO8601)
- Custom patterns: `toString('YYYY-MM-DD')` for date-only
- For display: `toString('DD.MM.YYYY hh:mm')`

**Date Control Binding:**
```tree
<= StartsAt_field $mol_form_field
    name \Starts At
    Content <= starts_at_control $mol_date
        value_moment? <=> starts_at_moment? null
```

```typescript
// In .view.ts file for GraphQL submission:
input: {
    startsAt: this.starts_at_moment()?.toString() || null,
}
```

## view.ts

In addition to declarative description of component, next to it could be created a file of the same name with `view.ts` extension, where a behavior could be described. Using a special construction, it could be inherited from realization obtained of `view.tree` and it would be overloaded automatically by heir:  

For example we have following description into `./my/hello/hello.view.tree`:

```tree
$my_hello $mol_view
	sub /
		<= Input $mol_string
			hint \Name
			value? <=> name? \
		<= message \
```

Here we have declared 2 properties: `name` to get the value from `Input` and `message` to output the value (we will override this property below).
It will be translated into following file `./my/hello/-view.tree/hello.view.tree.ts`: 

```typescript
namespace $ {
	export class $my_hello extends $mol_view {
		
		// sub /
		// 	<= Input
		// 	<= message
		sub() {
			return [
				this.Input(),
				this.message()
			] as readonly any[]
		}
		
		// name? \
		@ $mol_mem
		name(val?: any) {
			if ( val !== undefined ) return val as never
			return ""
		}
		
		// Input $mol_string
		// 	hint \Name
		// 	value? <=> name?
		@ $mol_mem
		Input() {
			const obj = new this.$.$mol_string()
			
			obj.hint = () => "Name"
			obj.value = (val?: any) => this.name(val)
			
			return obj
		}
		
		// message \
		message() { return "" }
	}
}
```

Now let's override the `message` method, which will use the `name` property in `./my/hello/hello.view.ts` behaivour file. `message` will depend on the `name` property entered by the user:

```typescript
namespace $.$$ {
	export class $my_hello extends $.$my_hello {
		
		message() {
			const name = this.name()
			return name && `Hello, ${name}!`
		}
		
	}
}
```
