# PDF

A `.pdf` printed from the HTML render, for a reader who files, prints or signs
what they receive. Read like the HTML, but fixed: nothing in it can be clicked
through to a section or reflowed on a phone.

## Markup

The HTML profile's, printed. Links may print as their text alone, so a URL the
reader needs is written out where it matters.

## Length

No cap. The core in full.

## Render

1. Make the HTML render, as the HTML profile says, to `<scratch>/render.html`
2. Find a printer, in order:
   - Chrome: `command -v google-chrome chromium`, else the macOS application
     at `/Applications/Google Chrome.app/Contents/MacOS/Google Chrome`
   - weasyprint: `command -v weasyprint`
3. Print to `<scratch>/render.pdf` with the first printer found, never both.
   Chrome:

   ```sh
   "<chrome>" --headless --no-pdf-header-footer \
     --print-to-pdf=<scratch>/render.pdf file://<scratch>/render.html
   ```

   weasyprint:

   ```sh
   weasyprint <scratch>/render.html <scratch>/render.pdf
   ```

   With neither printer, deliver the HTML instead, and the closing reply says a
   PDF could not be printed and why
4. Read the printed PDF's first page, Read taking a PDF's pages: a date, a
   path or a page number printed in its margin is Chrome's header, so the
   closing reply says it is there
