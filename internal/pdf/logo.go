package pdf

import (
	"bytes"
	"log/slog"

	"github.com/go-pdf/fpdf"
)

// LogoHeightMM is the fixed embed height of the EEG logo in PDFs. 30 mm
// at A4 lines up with the typical letterhead band — large enough to be
// recognisable, small enough not to dominate the first page.
const LogoHeightMM = 30.0

// LogoMaxWidthMM caps the horizontal footprint so a very wide logo
// (banner-style) gets scaled down proportionally instead of running
// across the page. fpdf preserves aspect ratio when only height is given,
// so we set height first; if the resulting width exceeds this we re-set
// with the width as the dominant dimension.
const LogoMaxWidthMM = 50.0

// fpdfImageType maps the MIME type returned by the eegFaktura-billing
// service to the short type tag fpdf wants in RegisterImageReader.
// Returns "" for anything outside the whitelist enforced at fetch time —
// caller should skip the embed in that case.
func fpdfImageType(mime string) string {
	switch mime {
	case "image/png":
		return "PNG"
	case "image/jpeg":
		return "JPG"
	case "image/gif":
		return "GIF"
	default:
		return ""
	}
}

// embedLogoCenteredRight draws the logo on the right edge of the page,
// vertically centered between topY and bottomY. Does NOT touch the
// cursor — callers render text first, capture the band's bottom Y
// (e.g. the header separator line), then call this to drop the logo
// into the empty right portion of that band. Used by the approval PDF.
//
// If the logo's height exceeds the band, it is anchored at topY (clipped
// from the bottom rather than spilling above the page top).
func embedLogoCenteredRight(f *fpdf.Fpdf, logoBytes []byte, mime string, topY, bottomY float64) {
	if len(logoBytes) == 0 || mime == "" {
		return
	}
	imgType := fpdfImageType(mime)
	if imgType == "" {
		slog.Warn("pdf: skipping logo embed — unsupported MIME", "mime", mime)
		return
	}
	const imgName = "eeg_logo"
	info := f.RegisterImageReader(imgName, imgType, bytes.NewReader(logoBytes))
	if f.Error() != nil {
		slog.Warn("pdf: logo registration failed; rendering without logo",
			"mime", mime, "bytes", len(logoBytes), "error", f.Error())
		f.ClearError()
		return
	}
	if info == nil {
		return
	}

	pageW, _ := f.GetPageSize()
	_, _, rm, _ := f.GetMargins()
	w := info.Width() * LogoHeightMM / info.Height()
	h := LogoHeightMM
	if w > LogoMaxWidthMM {
		w = LogoMaxWidthMM
		h = info.Height() * LogoMaxWidthMM / info.Width()
	}
	x := pageW - rm - w
	y := topY + (bottomY-topY-h)/2
	if y < topY {
		y = topY
	}

	// Preserve the cursor — caller has already rendered the band's text.
	cx, cy := f.GetX(), f.GetY()
	f.ImageOptions(imgName, x, y, w, h, false, fpdf.ImageOptions{ImageType: imgType}, 0, "")
	f.SetXY(cx, cy)

	if f.Error() != nil {
		slog.Warn("pdf: logo embed produced an fpdf error; clearing",
			"error", f.Error())
		f.ClearError()
	}
}

// embedLogoTopRight registers `logoBytes` as a named image and draws it
// in the top-right corner of the current page, preserving aspect ratio
// at LogoHeightMM. Restores the cursor position to (lm, current Y) before
// returning so callers can continue rendering without manual SetXY.
//
// On any failure (empty bytes, unsupported MIME, fpdf parse error) the
// function logs a warning and returns silently — PDF generation must NOT
// hard-fail because of a logo problem.
func embedLogoTopRight(f *fpdf.Fpdf, logoBytes []byte, mime string) {
	if len(logoBytes) == 0 || mime == "" {
		return
	}
	imgType := fpdfImageType(mime)
	if imgType == "" {
		slog.Warn("pdf: skipping logo embed — unsupported MIME", "mime", mime)
		return
	}

	// Stable per-PDF image name keyed by content hash would be ideal, but
	// for a single embed per document a single fixed name works — fpdf
	// only registers the image once per document anyway.
	const imgName = "eeg_logo"

	info := f.RegisterImageReader(imgName, imgType, bytes.NewReader(logoBytes))
	if f.Error() != nil {
		slog.Warn("pdf: logo registration failed; rendering without logo",
			"mime", mime, "bytes", len(logoBytes), "error", f.Error())
		// Reset the error so the rest of the document still renders.
		f.ClearError()
		return
	}
	if info == nil {
		return
	}

	// Compute draw position: top-right, anchored to the right margin.
	pageW, _ := f.GetPageSize()
	_, topMargin, rm, _ := f.GetMargins()

	// fpdf: ImageOptions(name, x, y, w, h, ...) — passing only h preserves
	// aspect ratio. If that produces a width > LogoMaxWidthMM, fall back to
	// width-driven scaling.
	w := info.Width() * LogoHeightMM / info.Height()
	h := LogoHeightMM
	if w > LogoMaxWidthMM {
		w = LogoMaxWidthMM
		h = info.Height() * LogoMaxWidthMM / info.Width()
	}
	x := pageW - rm - w
	y := topMargin
	f.ImageOptions(imgName, x, y, w, h, false, fpdf.ImageOptions{ImageType: imgType}, 0, "")

	// Reset cursor so the caller's subsequent CellFormat / Ln operations
	// start at the left margin, just below where they would have started
	// if the logo weren't there. We DO NOT push the cursor down by
	// LogoHeightMM — the title block is narrower than the logo so they
	// can coexist; pushing down would create a big blank strip.
	lm, _, _, _ := f.GetMargins()
	f.SetXY(lm, topMargin)

	if f.Error() != nil {
		slog.Warn("pdf: logo embed produced an fpdf error; clearing",
			"error", f.Error())
		f.ClearError()
		return
	}
}
