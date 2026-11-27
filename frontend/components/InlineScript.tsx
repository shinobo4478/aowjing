/**
 * An inline <script> that runs while the HTML is parsed (before first paint) on
 * the server, but is inert text when React renders it in the browser — React
 * warns about executable <script> tags in client renders, and they would not
 * run there anyway. See node_modules/next/dist/docs/01-app/02-guides/
 * preventing-flash-before-hydration.md.
 */
export default function InlineScript({ html }: { html: string }) {
  return (
    <script
      type={typeof window === "undefined" ? "text/javascript" : "text/plain"}
      suppressHydrationWarning
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}
