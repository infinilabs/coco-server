// Import the shared pipeline-ui standalone stylesheet BEFORE uno.css:
// it ships a full prebuilt utilities copy (.grid-cols-1, .md:grid-cols-2,
// ...). When bundled into a lazily loaded chunk it lands AFTER this app's
// utilities in the final CSS order, and its unscoped `.grid-cols-1`
// overrides same-specificity responsive variants like the home page's
// `md:grid-cols-3` — collapsing the AI Search/AI Chat/Knowledge base
// cards into one vertical column. Loading it first keeps the cascade:
// app utilities (uno.css) always win.
import '@infinilabs/pipeline-ui/styles.css';
import 'uno.css';
import 'virtual:svg-icons-register';
import '../styles/css/global.css';
