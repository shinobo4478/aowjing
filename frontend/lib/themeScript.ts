// Not a client module on purpose: the root layout (a Server Component) needs
// the script's text as a plain string, and values exported from a "use client"
// file reach a Server Component as references, not values.

export const THEME_STORAGE_KEY = "acmp-theme";

/**
 * Runs in <head> before first paint (see app/layout.tsx), so the page never
 * flashes the wrong colours. It must stay a plain string of old-style JS and
 * must resolve "system" the same way resolveMode() in app/ThemeMode.tsx does.
 */
export const THEME_INIT_SCRIPT = `(function(){try{var p=localStorage.getItem("${THEME_STORAGE_KEY}");var m=p==="light"||p==="dark"?p:(window.matchMedia("(prefers-color-scheme: light)").matches?"light":"dark");document.documentElement.setAttribute("data-theme",m)}catch(e){}})()`;
