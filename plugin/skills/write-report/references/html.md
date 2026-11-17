# HTML

A single `.html` file, opened in a browser, sent as an attachment or hosted as
a status page. It must render the same wherever it is opened, offline included,
so it is self-contained: CSS inline, no external font, script or image, no local
asset.

## Markup

What the Markdown profile carries, rendered: headings, tables, links and code.
No chart. A status page may lead with a short table of where each item stands.

## Length

No cap. The core in full.

## Render

1. Write the Markdown render first, as the Markdown profile says, to
   `<scratch>/render.md`
2. Write the style to `<scratch>/style.html`, a `<style>` block only: a
   system font stack, a readable measure of about 70 characters, table borders,
   and colours that hold in light and dark through `prefers-color-scheme`
3. Make the HTML with pandoc, without its `smart` extension, which would turn
   `...`, `--` and straight quotes typographic:

   ```sh
   pandoc --from=markdown-smart --to=html5 --standalone \
     --metadata pagetitle='<title>' --include-in-header=<scratch>/style.html \
     --output=<scratch>/render.html <scratch>/render.md
   ```

   `pagetitle`, not `title`, which would add a second `<h1>` over the render's
   own `#` title. The title is single-quoted for the shell, a `'` in it
   written `'\''`

   With no pandoc on `PATH`, write the HTML by hand from the Markdown render,
   the style inline in its `<head>`
4. Check it holds nothing external: grep for `<link`, `src=`, `@import` and
   `url(`, and judge each hit. A link, a script or an image the file would
   fetch or load from beside it is removed, and the render made again; text in
   the report that only quotes one of them stays
5. Count its characters for the brief: `wc -m < <scratch>/render.html`
