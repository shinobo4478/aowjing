// Date formatting that reads the same everywhere. `toLocaleString()` gave
// "9/3/2026, 7:04:00 PM" — ambiguous (3 Sep or 9 Mar?) and noisy.

const dateFmt = new Intl.DateTimeFormat("en-GB", {
  day: "numeric",
  month: "short",
  year: "numeric",
});

const dateTimeFmt = new Intl.DateTimeFormat("en-GB", {
  day: "numeric",
  month: "short",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
});

/** "3 Sep 2026" */
export const formatDate = (iso: string) => dateFmt.format(new Date(iso));

/** "3 Sep, 19:04" */
export const formatDateTime = (iso: string) => dateTimeFmt.format(new Date(iso));
