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

## Localization

The project localization process can follow two different approaches: through project resource files or through online web server requests. $mol uses the first approach by default.

### Localization in view.tree

Hardcoding texts in code is simple and fast, but what about localization? Simply add the `@` symbol and the text after it will be extracted to a separate file with translations, while the generated TypeScript class from view.tree will only contain a call by a human-readable key.

**Localization example:**
```tree
$trip2g_admin_telegrampublishnote_catalog $trip2g_admin_catalog
	menu_title @ \Telegram Publish Notes
	ShowSent $mol_check_box
		title <= show_sent_title @ \Show Sent ({count})
		checked? <=> show_sent? false
	Row_title_labeler* $mol_labeler
		title @ \Title
		Content <= Row_title* $trip2g_admin_cell
```

### Localization Process

The localization process consists of several steps:

1. **Add the `@` sign before the component property value** that needs to be translated into different languages. The translation key will be automatically generated from the source, and you can view it locally in the `component.view.tree.locale=en.json` file after running the project.

2. **Create a translation file in the component folder for each locale**. The file will have the same name as the component, but with a suffix.

**Example for Russian:** `component.view.tree.locale=ru.json`, where `ru` is the language code. The EN locale is automatically extracted from the component.

**Localization file structure:**
```json
{
	"$trip2g_admin_telegrampublishnote_catalog_show_sent_title": "Show Sent ({count})",
	"$trip2g_admin_telegrampublishnote_catalog_show_outdated_title": "Show Outdated ({count})",
	"$trip2g_admin_telegrampublishnote_catalog_Row_title_labeler_title": "Title",
	"$trip2g_admin_telegrampublishnote_catalog_Row_publish_at_labeler_title": "Publish At",
	"$trip2g_admin_telegrampublishnote_catalog_Row_status_labeler_title": "Status",
	"$trip2g_admin_telegrampublishnote_catalog_menu_title": "Telegram Posts"
}
```

`$trip2g_admin_telegrampublishnote_catalog_show_sent_title` is a key obtained according to the FQN component name (`$trip2g_admin_telegrampublishnote_catalog`) + property name (`show_sent_title`).

### Browser Usage

1. **Open the browser** with the application and developer console (F12 or Ctrl+Shift+I).

2. **Enter commands in the console** to change the project locale:
   - `$mol_locale.lang('en')` - English language
   - `$mol_locale.lang('ru')` - Russian language
   
   This command will change all localized texts on the site. If you enter a locale that doesn't exist `$mol_locale.lang('it')`, the default locale (en) will be applied.

3. **Current locale is stored in localStorage** of the site, so when the browser restarts, the user will retain their selected language.

4. **Get translations for all keys** on the site for the selected locale using the command `$mol_locale.texts('ru')`. This will include all texts, even those used in other components.

### Localization Workflow

When asked to localize a component, follow these steps:

1. **Mark text fields for localization** - Go into the component and mark fields like `title` with `@` symbol
2. **Check for existing Russian locale file** - Look for `component.view.tree.locale=ru.json` in the component folder
3. **Copy keys from generated English file** - After marking fields with `@`, the system generates `component/-/component.view.tree.locale=en.json`
4. **Transfer keys and translate** - Copy the keys from the English file to the Russian file and provide Russian translations

**Example workflow for `assets/ui/admin/telegrampublishnote/show/show.view.tree`:**

1. Mark titles with `@`: `title @ \Post Content`
2. Check if `assets/ui/admin/telegrampublishnote/show/show.view.tree.locale=ru.json` exists
3. Copy keys from `assets/ui/admin/telegrampublishnote/show/-/show.view.tree.locale=en.json`
4. Add Russian translations to the ru.json file

### Key Points

- **Automatic key generation**: Localization keys are automatically generated based on component FQN name and property
- **English fallback**: If translation is not found, English text from the component is used
- **Dynamic values**: Texts can use placeholders like `{count}` that are replaced at runtime
- **Locale hierarchy**: The system automatically searches for translations from more specific to more general locales

## Testing
- **Libraries**: `github.com/kr/pretty`, `github.com/matryer/moq`, `github.com/stretchr/testify/require`
- **Pattern**: Table-driven tests with mock setup functions
- **Mocks**: Generate with `//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env`