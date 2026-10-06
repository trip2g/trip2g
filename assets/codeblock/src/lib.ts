// highlight.js library bundle — wraps highlight.js and exposes it as
// window.hljs, mirroring how mermaid.min.js sets window.mermaid.
// Loaded lazily by codeblock.js only on pages that have code blocks.

import hljs from 'highlight.js/lib/common';
import type { HLJSApi, Language } from 'highlight.js';

// Jet templates (github.com/CloudyKit/jet): {{ actions }} and {* comments *}
// over HTML, which is handed to the xml grammar.
function jet(hljs: HLJSApi): Language {
  return {
    name: 'Jet',
    subLanguage: 'xml',
    contains: [
      { scope: 'comment', begin: /\{\*/, end: /\*\}/ },
      {
        beginScope: 'template-tag',
        endScope: 'template-tag',
        begin: /\{\{-?/,
        end: /-?\}\}/,
        keywords: {
          keyword: 'if else end range block yield import include extends return try catch',
          built_in: 'isset len map slice array ints exec includeIfExists json writeJson ' +
            'unsafe raw safeHtml safeJs html url js dump lower upper hasPrefix hasSuffix ' +
            'repeat replace split trimSpace',
          literal: 'true false nil',
        },
        contains: [
          hljs.QUOTE_STRING_MODE,
          hljs.APOS_STRING_MODE,
          { scope: 'string', begin: /`/, end: /`/ },
          hljs.C_NUMBER_MODE,
          { scope: 'variable', begin: /\.[A-Za-z_]\w*/ },
          { scope: 'operator', begin: /:=|==|!=|&&|\|\||[<>]=?|\|/ },
        ],
      },
    ],
  };
}

hljs.registerLanguage('jet', jet);

(window as any).hljs = hljs;
