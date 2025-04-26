# AI Context

This document summarizes the conventions and workflows for extending both backend (Go/SQL/GraphQL) and frontend (UI/$mol) in this project.

## Backend

1. **SQL Queries**  
   - All database queries live in `internal/db/queries.sql`.  
   - Use `-- name: MyNewQuery :one|many|exec` followed by standard SQL.  
   - Run `sqlc generate` after updating `queries.sql` to regenerate Go methods in `internal/db/queries.sql.go`.

2. **GraphQL Schema**  
   - Schemas are defined under `internal/graph/schema.graphqls`.  
   - When you add or modify types, inputs, or fields, run:  
     ```bash
     go run github.com/99designs/gqlgen generate
     ```
   - Resolver implementations live in `internal/graph/schema.resolvers.go` or in `internal/case/.../resolve.go`.

3. **Case Handlers**  
   - Business logic lives in `internal/case/...`.  
   - Each mutation or query has a `Request` type with a `Resolve(ctx, env)` method returning the GraphQL payload.  
   - The `Env` interface lists needed database methods (from `db.Queries`).

4. **Workflow**  
   - Add SQL definitions, regenerate with `sqlc`.  
   - Update GraphQL schema, regenerate with `gqlgen`.  
   - Implement or update `internal/case/.../resolve.go`.  
   - Write or update tests as needed.

## Frontend

1. **View Definitions**  
   - Tree-based UI specs live under `ui/` in `.view.tree` files, with behavior in corresponding `.view.ts`.  
   - List views use `$trip2g_graphql_request` + `$trip2g_graphql_make_map()`, then define `row(id)` and column getters.

2. **Routing & Linking**  
   - The main `ui/admin/admin.view.tree` uses `spreads` to switch pages by `nav` arg.  
   - List trees wire detail pages via `Content* $trip2g_admin_show_X` and `param \x_id <= row_id*`.

3. **Detail/Edit Pages**  
   - Show pages fetch a single record via GraphQL query in their `.view.ts`.  
   - Use `$mol_labeler`, `$mol_date`, `$mol_time_moment`, etc., for form controls.  
   - Bind inputs two-way using `<=>`, e.g., `value_moment? <=> expires_at_moment?`.

For details on the frontend framework, see **docs/mol.md**.  
