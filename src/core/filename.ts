/**
 * Source: New helper for CLI output naming.
 */
export function buildProgramFileName(title: string, ft: string): string {
  const date = ft.slice(0, 8);
  const safeTitle = sanitizeFileNamePart(title.trim() || "program");
  return `${safeTitle} - ${date}.aac`;
}

/**
 * Source: New helper for CLI output naming.
 */
export function sanitizeFileNamePart(value: string): string {
  const sanitized = value
    .replace(/[\\/:*?"<>|]/g, "_")
    .replace(/_+/g, "_")
    .replace(/\s+/g, " ")
    .trim();
  return sanitized === "" ? "program" : sanitized;
}
