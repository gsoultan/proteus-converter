package css

// UserAgentCSS is the default stylesheet, modelled on Chromium's html.css
// (print media, scripting enabled).
const UserAgentCSS = `
html, address, blockquote, body, center, dd, details, dialog, dir, div, dl, dt, fieldset,
figcaption, figure, footer, form, frameset, h1, h2, h3, h4, h5, h6, header, hgroup, hr,
legend, listing, main, menu, nav, ol, p, plaintext, pre, search, section, summary, ul, xmp,
article, aside, optgroup { display: block }
head, link, meta, script, style, title, template, noscript, datalist, param, area, base,
basefont, rp, [hidden], input[type=hidden], embed[hidden] { display: none }
body { margin: 8px }
p { margin-top: 1em; margin-bottom: 1em }
blockquote { margin: 1em 40px }
address, cite, dfn, em, i, var { font-style: italic }
center { text-align: center }
figure { margin: 1em 40px }
hr { color: gray; border-style: inset; border-width: 1px; margin: 0.5em auto; overflow: hidden }
h1 { font-size: 2em; margin-top: 0.67em; margin-bottom: 0.67em; font-weight: bold }
h2 { font-size: 1.5em; margin-top: 0.83em; margin-bottom: 0.83em; font-weight: bold }
h3 { font-size: 1.17em; margin-top: 1em; margin-bottom: 1em; font-weight: bold }
h4 { margin-top: 1.33em; margin-bottom: 1.33em; font-weight: bold }
h5 { font-size: 0.83em; margin-top: 1.67em; margin-bottom: 1.67em; font-weight: bold }
h6 { font-size: 0.67em; margin-top: 2.33em; margin-bottom: 2.33em; font-weight: bold }
table { display: table; border-collapse: separate; border-spacing: 2px; border-color: gray; box-sizing: border-box; text-indent: 0 }
caption { display: table-caption; text-align: center }
thead { display: table-header-group; vertical-align: middle; border-color: inherit }
tbody { display: table-row-group; vertical-align: middle; border-color: inherit }
tfoot { display: table-footer-group; vertical-align: middle; border-color: inherit }
col { display: table-column }
colgroup { display: table-column-group }
tr { display: table-row; vertical-align: inherit; border-color: inherit }
td, th { display: table-cell; vertical-align: inherit; padding: 1px }
th { font-weight: bold; text-align: center }
ul, menu, dir { list-style-type: disc; margin-top: 1em; margin-bottom: 1em; padding-left: 40px }
ol { list-style-type: decimal; margin-top: 1em; margin-bottom: 1em; padding-left: 40px }
li { display: list-item; text-align: match-parent }
ul ul, ol ul { list-style-type: circle }
ol ol ul, ol ul ul, ul ol ul, ul ul ul { list-style-type: square }
ul ul, ul ol, ol ul, ol ol, ul menu, ol menu, menu ul, menu ol { margin-top: 0; margin-bottom: 0 }
dd { margin-left: 40px }
dl { margin-top: 1em; margin-bottom: 1em }
dl dl, ol dl, ul dl { margin-top: 0; margin-bottom: 0 }
pre, xmp, plaintext, listing { font-family: monospace; white-space: pre; margin-top: 1em; margin-bottom: 1em }
code, kbd, samp, tt { font-family: monospace }
b, strong { font-weight: bolder }
u, ins { text-decoration: underline }
s, strike, del { text-decoration: line-through }
sub { vertical-align: sub; font-size: smaller }
sup { vertical-align: super; font-size: smaller }
small { font-size: smaller }
big { font-size: larger }
nobr { white-space: nowrap }
mark { background-color: mark; color: marktext }
a:link { color: -webkit-link; text-decoration: underline }
q::before { content: open-quote }
q::after { content: close-quote }
img { overflow: clip }
input, textarea, select, button { display: inline-block; margin: 0; font: 13.333px sans-serif }
input[type=checkbox], input[type=radio] { margin: 3px 3px 3px 4px; width: 13px; height: 13px; box-sizing: border-box }
textarea { white-space: pre-wrap }
fieldset { margin-left: 2px; margin-right: 2px; padding: 0.35em 0.75em 0.625em; border: 2px groove rgb(192,192,192) }
legend { padding-left: 2px; padding-right: 2px }
summary { display: block }
ruby { display: inline }
rt { font-size: 50% }
`
