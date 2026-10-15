# M6b: one pixel style

User (2026-10-15): "统一一下画风，都用像素风，不然有点割裂". Approved from `docs/design/mockups/v6.html`
(render `v6.png`): sections 1–3 and 5A. The app icon (section 4) waits: the user is still deciding.

- Interface icons: pixelarticons (MIT, `pixelarticons@2.4.2`, 24-unit viewBox drawn in 2-unit
  pixels, so a 12 × 12 pixel grid like the avatars). Vendor the SVGs used under `app/theme/pixel-icons/`
  with the LICENSE and a README line with the version; go:embed; tinted with the text color as the
  line icons are. Every current line icon gets its pixel twin (the pairs in v6.html section 1:
  inbox, calendar, checkbox-on for Completed, folder, gear, search, bulletlist, grid-3x2 for the board,
  check, bell, flag, arrow-right, copy, trash, clock, git-branch, terminal, sparkle, github, sliders,
  image, send, external-link, pencil, plus, close, chevron-down, arrow-up, and any other icon in use).
- Sizes: only whole multiples of the 12-pixel grid, so edges stay crisp: 12, 18, 24, 36 pt. Icons
  drawn at 14, 15, 16 or 20 pt today move to the nearest of those (menus and sidebar 18, small meta
  icons 12, Settings toolbar 24).
- Status glyphs: drawn on a 12 × 12 grid as in v6.html section 2 (stepped pixel circles; the check
  and the bang two pixels thick; colors unchanged): backlog, todo, doing, review, blocked, done,
  cancelled. The completing animation becomes two steps: the disc fills, then the check appears.
  Reduce Motion: straight to done.
- Unchanged: the system font, corner radii and shadows of the interface, the official provider
  logos (lobe-icons), the avatars.
