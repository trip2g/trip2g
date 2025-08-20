# Frontend Documentation

> **Note for Claude**: Read this documentation when working in the `assets/ui/` directory for frontend tasks.

## Admin CRUD Pages

The admin interface follows a consistent CRUD (Create, Read, Update, Delete) pattern for managing entities.

### Directory Structure

```
assets/ui/admin/[entity]/
├── catalog/                 # List view of all entities
├── show/                   # Detail view of single entity
├── update/                 # Edit form for entity
├── create/                 # Creation form (optional)
└── button/                 # Action buttons
    ├── run/
    ├── delete/
    └── refresh/
```

### Catalog Page Pattern

List all entities in a table format with links to detail pages.

**Key Features:**
- Extends `$trip2g_admin_catalog` which provides table layout
- Uses `$trip2g_graphql_request` to fetch data
- Converts data to map with `$trip2g_graphql_make_map` 
- Links to show pages via `ShowPage*` components

**Structure** (`catalog.view.tree`):
```tree
$trip2g_admin_[entity]_catalog $trip2g_admin_catalog
	menu_title \[Entity Name]
	ShowPage* $trip2g_admin_[entity]_show
		[entity]_id <= row_id* 0
	menu_link_content* / <= [Entity]Item* $mol_view
		sub /
			<= Rows* $mol_row
				sub <= row_content* /
					<= Row_id_labeler* $mol_labeler
						title \ID
						Content <= Row_id* $trip2g_admin_cell
```

**Business Logic** (`catalog.view.ts`):
```typescript
namespace $.$$ {
	export class $trip2g_admin_[entity]_catalog extends $.$trip2g_admin_[entity]_catalog {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_graphql_request(`query { admin { all[Entities] { nodes { id name } } } }`)
			return $trip2g_graphql_make_map(res.admin.all[Entities].nodes)
		}

		row(id: any) { return this.data().get(id) }
		override row_id(id: any): number { return this.row(id).id }
	}
}
```

### Show Page Pattern

Display detailed information about a single entity with edit/action buttons.

**Structure** (`show.view.tree`):
```tree
$trip2g_admin_[entity]_show $mol_page
	[entity]_id? 0
	tools /
		<= EditLink $mol_link
			arg * action \update
		<= ActionButton $mol_button_major
	body /
		<= Details $mol_view
			sub /
				<= Id_labeler $mol_labeler
					title \ID
					Content <= Id $trip2g_admin_cell
```

**Business Logic** (`show.view.ts`):
```typescript
namespace $.$$ {
	export class $trip2g_admin_[entity]_show extends $.$trip2g_admin_[entity]_show {
		@$mol_mem
		[entity]_data(reset?: null) {
			const res = $trip2g_graphql_request(`query($id: Int64!) { admin { [entity](id: $id) { id name } } }`, { id: this.[entity]_id() })
			return res.admin.[entity]
		}
	}
}
```

### Update Form Pattern

Provide form interface for editing entity properties.

**Structure** (`update.view.tree`):
```tree
$trip2g_admin_[entity]_update $mol_view
	[entity]_id 0
	sub /
		<= Form $mol_form
			body /
				<= Field_field $mol_form_field
					Content <= field_control $mol_string
						value? <=> field? \
			buttons /
				<= Submit $mol_button_major
					click? <=> submit? null
				<= Result $mol_status
					message <= result? \
```

### Action Button Pattern

**IMPORTANT**: All action buttons must be separated into their own components under `button/[action]/`.

**Structure** (`button/[action]/[action].view.tree`):
```tree
$trip2g_admin_[entity]_button_[action] $mol_button_major
	[entity]_id 0
	title <= status_title? \[Action]
	click? <=> [action]? null
```

**Logic** (`button/[action]/[action].view.ts`):
```typescript
namespace $.$$ {
	export class $trip2g_admin_[entity]_button_[action] extends $.$trip2g_admin_[entity]_button_[action] {
		[action](event?: Event) {
			const res = $trip2g_graphql_request(`mutation($input: [Action][Entity]Input!) { admin { [action][Entity](input: $input) { ... on [Action][Entity]Payload { success } ... on ErrorPayload { message } } } }`, { input: { id: this.[entity]_id() } })
			
			if(res.admin.[action][Entity].__typename === 'ErrorPayload') {
				throw new Error(res.admin.[action][Entity].message)
			}
			this.status_title('[Action]: Success')
		}
	}
}
```

### CSS Styling

**Column Width Guidelines:**
- **ID columns**: `rem(3)` 
- **Dates/timestamps**: `rem(8)`
- **Names/titles**: `rem(12)`
- **Descriptions**: `rem(15)`

**Example** (`catalog.view.css.ts`):
```typescript
namespace $.$$ {
	const { rem } = $mol_style_unit

	$mol_style_define($trip2g_admin_[entity]_catalog, {
		Row_id_labeler: { flex: { basis: rem(3) } },
		Row_name_labeler: { flex: { basis: rem(12) } },
	})
}
```

## Mol Framework Essentials

### View Definitions
- Tree-based UI specs in `.view.tree` files, behavior in `.view.ts`
- List views use `$trip2g_graphql_request` + `$trip2g_graphql_make_map()`, then define `row(id)` methods

### Routing & Linking
- Main admin uses `spreads` to switch pages by `nav` arg
- Wire detail pages via `Content* $trip2g_admin_show_X` and `param \x_id <= row_id*`

### Detail/Edit Pages
- Show pages fetch single records via GraphQL query
- Use `$mol_labeler`, `$mol_date`, `$mol_time_moment` for form controls
- Bind inputs two-way using `<=>`, e.g., `value_moment? <=> expires_at_moment?`

### GraphQL Requests
- Use `$trip2g_graphql_request` for all queries/mutations
- Run `npm run graphqlgen` after modifying schema

## Date/Time Formatting

### $mol_time_moment

**Date Input Controls:**
- Use `$mol_date` for date inputs: `value_moment? <=> date_moment? null`

**Time/DateTime Formatting for GraphQL:**
- Use `moment.toString()` for Go backend (ISO8601 format)
- **Date-only inputs**: Convert to full datetime:
  ```typescript
  startsAt: this.starts_at_moment() ? new $mol_time_moment(this.starts_at_moment().toString() + 'T00:00:00Z').toString() : null
  ```
- **DateTime inputs**: `createdAt: this.created_at_moment()?.toString() || null`

**Patterns:**
- `toString()` produces `YYYY-MM-DDThh:mm:ss.sssZ` (ISO8601)
- Custom patterns: `toString('YYYY-MM-DD')` for date-only
- For display: `toString('DD.MM.YYYY hh:mm')`

## Testing
- **Libraries**: `github.com/kr/pretty`, `github.com/matryer/moq`, `github.com/stretchr/testify/require`
- **Pattern**: Table-driven tests with mock setup functions
- **Mocks**: Generate with `//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env`