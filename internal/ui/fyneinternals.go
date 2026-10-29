package ui

// Fyne v2.8 has no public way to reach a Table's scroller or to read its
// column widths. The wheel filter (FR-020) and the Name column fill need
// both, so they are read here via reflection. Everything degrades to "not
// available" if Fyne changes these internals; fyneinternals_test.go fails
// loudly when that happens.

import (
	"reflect"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// tableScroller returns the table's internal scroll container and the
// content it scrolls, or ok == false before the table is rendered or if
// Fyne's internals changed.
func tableScroller(t *widget.Table) (scroll fyne.Scrollable, obj, content fyne.CanvasObject, ok bool) {
	f := reflect.ValueOf(t).Elem().FieldByName("content")
	if !f.IsValid() || f.Kind() != reflect.Pointer || f.IsNil() {
		return nil, nil, nil, false
	}
	v := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem().Interface()
	scroll, ok1 := v.(fyne.Scrollable)
	obj, ok2 := v.(fyne.CanvasObject)
	c := reflect.ValueOf(v).Elem().FieldByName("Content")
	if !ok1 || !ok2 || !c.IsValid() || c.IsNil() {
		return nil, nil, nil, false
	}
	content, ok3 := c.Interface().(fyne.CanvasObject)
	return scroll, obj, content, ok3
}

// tableColumnWidth returns the current width of column col, which the user
// may have changed by dragging, or ok == false if it is not known.
func tableColumnWidth(t *widget.Table, col int) (float32, bool) {
	f := reflect.ValueOf(t).Elem().FieldByName("columnWidths")
	if !f.IsValid() || f.Kind() != reflect.Map || f.IsNil() {
		return 0, false
	}
	w := f.MapIndex(reflect.ValueOf(col))
	if !w.IsValid() || !w.CanFloat() {
		return 0, false
	}
	return float32(w.Float()), true
}
