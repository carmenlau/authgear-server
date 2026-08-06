// ICU MessageFormat treats single quotes (') as the start of a quoted literal
// section and curly braces ({}) as argument placeholders. Plain-text values
// containing these characters must be escaped before being stored as message
// patterns.
//
// Escape order: apostrophes first, then braces (so the delimiter quotes we
// introduce for { and } are never themselves misread as user apostrophes).
// Unescape order is the exact reverse.
export function escapeMessageFormatText(text: string): string {
  return text
    .replace(/'/g, "''")
    .replace(/\{/g, "'{'")
    .replace(/\}/g, "'}'");
}

export function unescapeMessageFormatText(text: string): string {
  return text
    .replace(/'}'/g, "}")
    .replace(/'\{'/g, "{")
    .replace(/''/g, "'");
}
