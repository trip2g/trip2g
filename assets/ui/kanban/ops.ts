namespace $ {
	function $trip2g_kanban_clone(b: $trip2g_kanban_board): $trip2g_kanban_board {
		return {
			frontmatter: b.frontmatter,
			settings: b.settings,
			lists: b.lists.map(l => ({ ...l, cards: l.cards.map(c => ({ ...c })) })),
		}
	}

	export function $trip2g_kanban_move_card(
		b: $trip2g_kanban_board,
		spec: { from: number; card: number; to: number; at: number },
	): $trip2g_kanban_board {
		const next = $trip2g_kanban_clone(b)
		const [card] = next.lists[spec.from].cards.splice(spec.card, 1)
		if (!card) return b
		const at = Math.max(0, Math.min(spec.at, next.lists[spec.to].cards.length))
		next.lists[spec.to].cards.splice(at, 0, card)
		return next
	}

	export function $trip2g_kanban_add_card(
		b: $trip2g_kanban_board,
		list: number,
		text: string,
	): $trip2g_kanban_board {
		const next = $trip2g_kanban_clone(b)
		next.lists[list].cards.push({ text, checked: false })
		return next
	}

	export function $trip2g_kanban_edit_card(
		b: $trip2g_kanban_board,
		list: number,
		card: number,
		patch: Partial<$trip2g_kanban_card>,
	): $trip2g_kanban_board {
		const next = $trip2g_kanban_clone(b)
		next.lists[list].cards[card] = { ...next.lists[list].cards[card], ...patch }
		return next
	}

	export function $trip2g_kanban_delete_card(
		b: $trip2g_kanban_board,
		list: number,
		card: number,
	): $trip2g_kanban_board {
		const next = $trip2g_kanban_clone(b)
		next.lists[list].cards.splice(card, 1)
		return next
	}
}
