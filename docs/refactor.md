  High Impact Refactoring Opportunities

  1. Extract App Configuration (cmd/server/main.go:79-127)
  - Move appConfig struct and flag setup to separate config package
  - 100+ line main() function should be broken down

  2. Separate Database Setup Logic (cmd/server/main.go:218-246)
  - Extract database initialization, pragmas, and migration logic into internal/db/setup.go

  3. Extract Admin Authorization Helper (schema.resolvers.go:285-291)
  - Multiple admin resolvers duplicate identical auth checks
  - Create internal/auth/admin.go with RequireAdmin() helper

  4. Consolidate Active Offer Query Patterns (queries.sql:195-214, 727-778)
  - Repeated active offer conditions across multiple queries
  - Create SQL view or helper method

  5. Refactor Server Handler Function (cmd/server/main.go:886-995)
  - 100+ line request handler with nested conditionals
  - Extract middleware and route handling

  6. Extract Transaction Management (cmd/server/main.go:350-412)
  - AcquireTxEnvInRequest and ReleaseTxEnvInRequest should be in separate package
  - Complex transaction logic mixed with app logic

  7. Standardize Error Handling in Case Handlers
  - Inconsistent patterns between ozzo validation and manual validation
  - Create common error handling utilities

  8. Extract Asset Management (cmd/server/main.go:414-454)
  - Asset URL generation and filesystem setup should be separate package

  9. Consolidate User Ban Logic (cmd/server/main.go:660-695)
  - Complex caching logic should be extracted to internal/cache/userbans.go

  10. Simplify Database Hash Collision Logic (internal/db/queries.go:20-86)
  - InsertNote method has complex collision resolution
  - Extract to separate hash generation package

  Medium Impact Opportunities

  11. Create Nullable Type Helpers (internal/db/helpers.go:18-91)
  - Multiple ToNullable* functions with repetitive patterns
  - Use generics for single ToNullable[T] function

  12. Extract Purchase Notification System (cmd/server/main.go:486-535)
  - Complex subscription management mixed with app struct
  - Move to separate internal/notifications package

  13. Standardize resolveOne Pattern (schema.resolvers.go:37-42)
  - Repeated pattern throughout GraphQL resolvers
  - Extract to common resolver utilities

  14. Consolidate Active User Access Queries (queries.sql:98-105, 265-272)
  - Similar filtering patterns for user subgraph access
  - Create SQL view for active accesses

  15. Extract String Generation Utilities (cmd/server/main.go:743-782)
  - GenerateApiKey and GeneratePurchaseID use similar patterns
  - Create internal/generate package

  16. Simplify Router Implementation (internal/router/router.go:77-144)
  - Large handle method with repeated error marshaling
  - Extract middleware pattern

  17. Consolidate Time-Based Filters (queries.sql)
  - Repeated datetime filtering patterns
  - Create parameterized time filter helpers

  18. Extract GraphQL Resolver Boilerplate (schema.resolvers.go:516-646)
  - 30+ resolver factory methods
  - Generate or simplify resolver registration

  19. Separate Environment Variables (cmd/server/main.go:784-790)
  - Environment variable access scattered throughout
  - Create config package with typed getters

  20. Consolidate Note Asset Queries (queries.sql:288-303)
  - Two nearly identical note asset lookup queries
  - Merge or create one as alias of other
