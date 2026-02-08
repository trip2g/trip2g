import { Editor, rootCtx, defaultValueCtx, editorViewCtx, serializerCtx, parserCtx } from "@milkdown/kit/core";
import { commonmark } from "@milkdown/kit/preset/commonmark";
import { listener, listenerCtx } from "@milkdown/kit/plugin/listener";
import { $remark } from "@milkdown/kit/utils";
import { Slice } from "@milkdown/kit/prose/model";
import remarkWikiLink from "remark-wiki-link";

const wikiLinkRemark = $remark("wikiLink", () =>
	remarkWikiLink({
		aliasDivider: "|",
		hrefTemplate: (permalink: string) => `/${permalink}`,
		wikiLinkClassName: "wikilink",
		newClassName: "wikilink-new",
	})
);

export interface MilkdownInstance {
	create(root: HTMLElement, defaultValue?: string): Promise<void>;
	destroy(): Promise<void>;
	getMarkdown(): string;
	setMarkdown(markdown: string): void;
	onChange(callback: (markdown: string) => void): void;
}

export function createMilkdown(): MilkdownInstance {
	let editor: Editor | null = null;
	let changeCallback: ((markdown: string) => void) | null = null;

	return {
		async create(root: HTMLElement, defaultValue = "") {
			editor = await Editor.make()
				.config((ctx) => {
					ctx.set(rootCtx, root);
					ctx.set(defaultValueCtx, defaultValue);
					ctx.set(listenerCtx, {
						markdown: [(getMarkdown) => {
							if (changeCallback) {
								changeCallback(getMarkdown());
							}
						}],
					});
				})
				.use(commonmark)
				.use(wikiLinkRemark)
				.use(listener)
				.create();
		},

		async destroy() {
			if (editor) {
				await editor.destroy();
				editor = null;
			}
		},

		getMarkdown(): string {
			if (!editor) return "";
			return editor.action((ctx) => {
				const view = ctx.get(editorViewCtx);
				const serializer = ctx.get(serializerCtx);
				return serializer(view.state.doc);
			});
		},

		setMarkdown(markdown: string) {
			if (!editor) return;
			editor.action((ctx) => {
				const view = ctx.get(editorViewCtx);
				const parser = ctx.get(parserCtx);
				const doc = parser(markdown);
				if (!doc) return;
				const state = view.state;
				view.dispatch(
					state.tr.replace(
						0,
						state.doc.content.size,
						new Slice(doc.content, 0, 0)
					)
				);
			});
		},

		onChange(callback: (markdown: string) => void) {
			changeCallback = callback;
		},
	};
}
