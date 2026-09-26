// Focuses its node once, when it is created — replaces the HTML `autofocus` attribute (a11y
// warning), which fires on parse, may steal focus already moved elsewhere and lets the browser
// pick among several. Usage: `<input use:autofocus />`.
export function autofocus(node) {
    node.focus();
}
