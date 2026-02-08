import { Crepe, CrepeFeature } from "@milkdown/crepe";
import { $remark } from "@milkdown/kit/utils";
import { Slice } from "@milkdown/kit/prose/model";
import { editorViewCtx, serializerCtx, parserCtx } from "@milkdown/kit/core";
import remarkWikiLink from "remark-wiki-link";

import "@milkdown/crepe/theme/common/style.css";
import "@milkdown/crepe/theme/frame.css";

const wikiLinkRemark = $remark("wikiLink", () => remarkWikiLink, {
	aliasDivider: "|",
	hrefTemplate: (permalink: string) => `/${permalink}`,
	wikiLinkClassName: "wikilink",
	newClassName: "wikilink-new",
});

export interface MilkdownInstance {
	create(root: HTMLElement, defaultValue?: string): Promise<void>;
	destroy(): Promise<void>;
	getMarkdown(): string;
	setMarkdown(markdown: string): void;
	onChange(callback: (markdown: string) => void): void;
}

export function createMilkdown(): MilkdownInstance {
	let crepe: Crepe | null = null;
	let changeCallback: ((markdown: string) => void) | null = null;

	return {
		async create(root: HTMLElement, defaultValue = "") {
			crepe = new Crepe({
				root,
				defaultValue,
				features: {
					[CrepeFeature.Latex]: false,
					[CrepeFeature.CodeMirror]: false,
					[CrepeFeature.ImageBlock]: false,
					[CrepeFeature.Table]: false,
				},
			});

			crepe.editor.use(wikiLinkRemark);

			if (changeCallback) {
				crepe.on((listener) => {
					listener.markdownUpdated((_ctx, markdown) => {
						changeCallback!(markdown);
					});
				});
			}

			await crepe.create();
		},

		async destroy() {
			if (crepe) {
				await crepe.destroy();
				crepe = null;
			}
		},

		getMarkdown(): string {
			if (!crepe) return "";
			return crepe.getMarkdown();
		},

		setMarkdown(markdown: string) {
			if (!crepe) return;
			crepe.editor.action((ctx) => {
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
