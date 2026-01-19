# Admin CRUD Step-by-Step Guide

Step-by-step guide for creating a complete CRUD interface for a new entity.

## Prerequisites

Before starting frontend work, ensure backend is ready:
1. Database table exists (migration applied)
2. SQL queries in `queries.read.sql` / `queries.write.sql`
3. GraphQL schema types and mutations defined
4. Resolvers implemented

## Steps Overview

1. Create catalog (list view)
2. Create show page (detail view)
3. Create delete confirmation
4. Add GraphQL query for single item
5. Wire everything together

## Step 1: Create Catalog

Directory: `assets/ui/admin/[entity]/catalog/`

### catalog.view.tree

```tree
$trip2g_admin_[entity]_catalog $trip2g_admin_catalog
	menu_title @ \[Entity Name]
	actions /
		<= AddLink $mol_link
			arg * id \add
			title \+ Add
	param \id
	Empty $mol_status
	ShowPage* $trip2g_admin_[entity]_show
		[entity]_id <= row_id* 0
	menu_link_content* / <= [Entity]Item* $mol_view
		sub /
			<= Rows* $mol_row
				sub <= row_content* /
					<= Id* $trip2g_admin_labeler_id
						value <= row_id_string* \id
					<= Name* $mol_labeler
						title \Name
						Content <= NameCell* $trip2g_admin_cell
							content <= row_name* \name
	AddForm $trip2g_admin_[entity]_create
```

### catalog.view.ts

```typescript
namespace $.$$ {
	const list_query = $trip2g_graphql_request(/* GraphQL */ `
		query Admin[Entity]List {
			admin {
				all[Entities] {
					nodes {
						id
						name
						createdAt
					}
				}
			}
		}
	`)

	export class $trip2g_admin_[entity]_catalog extends $.$trip2g_admin_[entity]_catalog {
		@$mol_mem
		data( reset?: null ) {
			return $trip2g_graphql_make_map( list_query().admin.all[Entities].nodes )
		}

		@$mol_mem
		spreads(): any {
			return {
				add: this.AddForm(),
				...this.data().mapKeys( key => this.Content( key ) ),
			}
		}

		@$mol_mem
		override spread_ids_filtered() {
			return this.spread_ids().filter( id => id !== 'add' )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		row_id_string( id: any ) {
			return this.row( id ).id.toString()
		}

		row_name( id: any ) {
			return this.row( id ).name
		}
	}
}
```

## Step 2: Create Show Page

Directory: `assets/ui/admin/[entity]/show/`

### show.view.tree

```tree
$trip2g_admin_[entity]_show $mol_page
	[entity]_id? 0
	title \[Entity Name]
	tools /
		<= DeleteLink $mol_link
			arg * action \delete
			title \Delete
	DeleteForm $trip2g_admin_[entity]_delete
		[entity]_id <= [entity]_id
	body /
		<= Details $mol_view
			sub /
				<= Details_row $mol_view
					style *
						flexDirection \column
					sub /
						<= Id_labeler $mol_labeler
							title \ID
							Content <= Id $trip2g_admin_cell
								content <= [entity]_id_string \
						<= Name_labeler $mol_labeler
							title \Name
							Content <= Name $trip2g_admin_cell
								content <= [entity]_name \
						<= CreatedAt_labeler $mol_labeler
							title \Created At
							Content <= CreatedAt $trip2g_admin_cell
								content <= [entity]_created_at \
```

**Note**: Use `$mol_view` with `style * flexDirection \column` for vertical layout instead of `$mol_row`.

### show.view.ts

```typescript
namespace $.$$ {
	const query = $trip2g_graphql_request(/* GraphQL */ `
		query Admin[Entity]ById($id: Int!) {
			admin {
				[entity](id: $id) {
					id
					name
					createdAt
				}
			}
		}
	`)

	export class $trip2g_admin_[entity]_show extends $.$trip2g_admin_[entity]_show {
		action() {
			return this.$.$mol_state_arg.value( 'action' ) || 'view'
		}

		@$mol_mem
		data() {
			return query({ id: this.[entity]_id() }).admin.[entity]
		}

		override body() {
			if( this.action() === 'delete' ) {
				return [ this.DeleteForm() ]
			}

			return super.body()
		}

		[entity]_id_string() {
			return String( this.data().id )
		}

		[entity]_name() {
			return this.data().name
		}

		[entity]_created_at() {
			return this.data().createdAt
		}
	}
}
```

## Step 3: Create Delete Confirmation

Directory: `assets/ui/admin/[entity]/delete/`

### delete.view.tree

```tree
$trip2g_admin_[entity]_delete $mol_view
	[entity]_id 0
	style *
		flexDirection \column
	sub /
		<= Confirm_text $mol_paragraph
			title \Are you sure you want to delete this [entity]?
		<= Delete_button $mol_button_major
			title \Yes, delete
			click? <=> delete? null
```

### delete.view.ts

```typescript
namespace $.$$ {
	const delete_mutation = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminDelete[Entity]($input: Delete[Entity]Input!) {
			admin {
				data: delete[Entity](input: $input) {
					__typename
					... on ErrorPayload {
						message
					}
					... on Delete[Entity]Payload {
						deletedId
					}
				}
			}
		}
	`)

	export class $trip2g_admin_[entity]_delete extends $.$trip2g_admin_[entity]_delete {
		delete() {
			const res = delete_mutation({
				input: { id: this.[entity]_id() },
			})

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.data.message )
			}

			if( res.admin.data.__typename === 'Delete[Entity]Payload' ) {
				this.$.$mol_state_arg.value( 'id', null )
				this.$.$mol_state_arg.value( 'action', null )
			}
		}
	}
}
```

## Step 4: Create Form (Create/Update)

Directory: `assets/ui/admin/[entity]/create/`

### create.view.tree

```tree
$trip2g_admin_[entity]_create $mol_view
	sub /
		<= Form $mol_form
			body /
				<= Name_field $mol_form_field
					name \name
					Content <= Name_control $mol_string
						hint \Enter name
						value? <=> name? \
			buttons /
				<= Submit $mol_button_major
					title \Create
					click? <=> submit? null
				<= Result $mol_status
					message <= result_message? \
```

### create.view.ts

```typescript
namespace $.$$ {
	const create_mutation = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminCreate[Entity]($input: Create[Entity]Input!) {
			admin {
				data: create[Entity](input: $input) {
					__typename
					... on ErrorPayload {
						message
						byFields { name value }
					}
					... on Create[Entity]Payload {
						[entity] { id }
					}
				}
			}
		}
	`)

	export class $trip2g_admin_[entity]_create extends $.$trip2g_admin_[entity]_create {
		submit() {
			const res = create_mutation({
				input: {
					name: this.name(),
				},
			})

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				if( res.admin.data.byFields?.length ) {
					const errors = res.admin.data.byFields.map( f => `${f.name}: ${f.value}` ).join( ', ' )
					throw new Error( errors )
				}
				throw new Error( res.admin.data.message )
			}

			if( res.admin.data.__typename === 'Create[Entity]Payload' ) {
				this.$.$mol_state_arg.value( 'id', String( res.admin.data.[entity].id ) )
			}
		}
	}
}
```

## Step 5: Add GraphQL Query for Single Item

In `internal/graph/schema.graphqls`, add query to AdminQuery:

```graphql
type AdminQuery {
  # ...existing queries...
  [entity](id: Int!): Admin[Entity]
}
```

Run `make gqlgen` and implement resolver in `schema.resolvers.go`:

```go
func (r *adminQueryResolver) [Entity](ctx context.Context, obj *appmodel.AdminQuery, id int32) (*db.[Entity], error) {
	item, err := r.env(ctx).Get[Entity](ctx, int64(id))
	if err != nil {
		return nil, err
	}
	return &item, nil
}
```

## Step 6: Register in Admin Navigation

In `assets/ui/admin/admin.view.tree`, add catalog:

```tree
$trip2g_admin $mol_page
	sub /
		<= Nav $mol_nav
			links /
				# ...existing links...
				<= [Entity]Link $mol_link
					arg * page \[entity]
					title \[Entities]
	spreads *
		# ...existing spreads...
		[entity] <= [Entity]Catalog $trip2g_admin_[entity]_catalog
```

## Checklist

- [ ] Catalog with list query
- [ ] Show page with single item query
- [ ] Delete confirmation form
- [ ] Create form (if needed)
- [ ] Update form (if needed)
- [ ] GraphQL query for single item
- [ ] Resolver implemented
- [ ] Added to admin navigation
- [ ] Run `npm run graphqlgen` for frontend types

## Common Patterns

### Action via URL Parameter

Show page handles different actions via URL:

```typescript
action() {
	return this.$.$mol_state_arg.value( 'action' ) || 'view'
}

override body() {
	if( this.action() === 'delete' ) return [ this.DeleteForm() ]
	if( this.action() === 'update' ) return [ this.UpdateForm() ]
	return super.body()
}
```

### Navigate After Action

```typescript
// Go back to list
this.$.$mol_state_arg.value( 'id', null )
this.$.$mol_state_arg.value( 'action', null )

// Go to created item
this.$.$mol_state_arg.value( 'id', String( res.admin.data.[entity].id ) )
```

### Error Handling

```typescript
if( res.admin.data.__typename === 'ErrorPayload' ) {
	// Field-level errors
	if( res.admin.data.byFields?.length ) {
		const errors = res.admin.data.byFields.map( f => `${f.name}: ${f.value}` ).join( ', ' )
		throw new Error( errors )
	}
	// General error
	throw new Error( res.admin.data.message )
}
```
