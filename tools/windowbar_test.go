package main

import (
	"image"
	"testing"
)

func TestWindowBarRevealAndPin(t *testing.T) {
	b := newWindowBar()
	if b.reveal(false, image.Pt(20, 6)) || b.Visible {
		t.Fatal("must start hidden outside top six pixels")
	}
	if !b.reveal(false, image.Pt(20, 5)) || !b.Visible {
		t.Fatal("top edge must reveal")
	}
	b.Menu = 0
	if b.reveal(false, image.Pt(20, 79)) || !b.Visible {
		t.Fatal("dropdown must hold reveal")
	}
	if !b.reveal(false, image.Pt(20, 80)) || b.Visible {
		t.Fatal("leaving dropdown must hide")
	}
	if !b.reveal(true, image.Pt(-1, -1)) || !b.Pinned {
		t.Fatal("escape must pin")
	}
	if b.reveal(false, image.Pt(-1, -1)) || !b.Visible {
		t.Fatal("pinned bar must persist")
	}
	b.reveal(true, image.Pt(20, 0))
	if b.reveal(false, image.Pt(20, 0)) || b.Visible {
		t.Fatal("must not reopen under cursor after escape")
	}
	b.reveal(false, image.Pt(20, 200))
	if !b.reveal(false, image.Pt(20, 0)) {
		t.Fatal("must reopen after leaving top")
	}
	b.close()
	if b.reveal(false, image.Pt(20, 0)) || b.Visible {
		t.Fatal("close click must not reopen")
	}
}

func TestWindowBarGeometry(t *testing.T) {
	b := newWindowBar()
	b.Visible = true
	b.Menu = 1
	for _, c := range []struct {
		p    image.Point
		f, r int
	}{
		{image.Pt(200, 32), 1, 0}, {image.Pt(367, 79), 1, 1}, {image.Pt(368, 79), -1, -1},
		{image.Pt(600, 4), 3, -1}, {image.Pt(632, 4), -1, -1}, {image.Pt(8, 28), -1, -1},
	} {
		f, r := b.hit(c.p)
		if f != c.f || r != c.r {
			t.Fatalf("%v: %d %d", c.p, f, r)
		}
	}
	for _, c := range []struct {
		w, h    int
		visible bool
		delta   int
	}{
		{640, 400, false, 32}, {1280, 800, false, 64}, {1280, 864, true, 64}, {1000, 700, false, 50}, {1000, 750, true, 50},
	} {
		if got := barHeightDelta(c.w, c.h, c.visible); got != c.delta {
			t.Fatalf("height delta %d != %d", got, c.delta)
		}
	}
	if p, ok := barGamePoint(600, 406, true); !ok || p != image.Pt(300, 171) {
		t.Fatal(p, ok)
	}
	if _, ok := barGamePoint(600, 63, true); ok {
		t.Fatal("toolbar must stay outside game")
	}
	if _, ok := barGamePoint(-1, 64, true); ok {
		t.Fatal("negative edge must be rejected before division")
	}
	if _, ok := barGamePoint(1280, 864, true); ok {
		t.Fatal("exclusive edge")
	}
}

func TestWindowBarFontAndPaint(t *testing.T) {
	f, err := loadBarFont("/project/tools/toolbar-font.hex.gz")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range barLabels {
		for _, r := range s {
			if f[r].Width == 0 {
				t.Fatal("missing", string(r))
			}
		}
	}
	for _, list := range barLists {
		for _, s := range list {
			for _, r := range s {
				if f[r].Width == 0 {
					t.Fatal("missing", string(r))
				}
			}
		}
	}
	b := newWindowBar()
	b.Visible = true
	b.Menu = 0
	b.Hover = 1
	p := b.paint(f, [3]int{}, true)
	if p.NRGBAAt(0, 79).A != 0 || p.NRGBAAt(10, 79).A != 255 {
		t.Fatal("dropdown must cover its own field only")
	}
	if _, err = loadBarFont("/project/tools/no-such-font.gz"); err == nil {
		t.Fatal("missing font must be explicit")
	}
}
