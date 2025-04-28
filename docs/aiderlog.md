/add docs/aicontext.md ui/admin/button/user/unban/unban.view.tree ui/admin/button/user/unban/unban.view.ts queries.sql internal/graph/schema.graphqls internal/case/admin/unbanuser/resolve.go internal/graph/resolver.go internal/graph/schema.resolvers.go ui/admin/list/users/users.view.tree internal/db/queries.sql.go
Based on ui/admin/button/user/unban/* you need make ui/admin/button/user/ban/* and add it to ui/admin/list/users/users.view.tree. Before that you need make a graphql mutation based on internal/case/admin/unbanuser/resolve.go and add it to internal/graph/schema.resolvers.go. Before that you need write this mutation to internal/graph/schema.graphqls and run `go run github.com/99designs/gqlgen generate`. Also you need write SQL queries in queries.sql and run `sqlc gen`. Check signaturs of queries in internal/db/queries.sql.go. Also you need add a case Env to resolver.go.

### Нужно добавить бан пользователя

Для этого нужно отдельную страницу сделать, думаю
ui/admin/show/bunuser/* с формой с одним полем reason.

Примеры формы можно взять из
ui/admin/select/subgraph/subgraph.view.ts
ui/admin/select/subgraph/subgraph.view.tree

Только вместо submit будет кнорка мутации бана.

Нужно будет добавить кнорку открытия этой страницы в
ui/admin/list/users/users.view.tree и так же новый роут в
ui/admin/admin.view.tree
