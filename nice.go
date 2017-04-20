package sketchmerge

import (
	"fmt"
)

func GetNiceTextForUnknown(srcact ApplyAction, key string) (string, string) {

	var niceDesc string = ""
	var niceDescShort string = ""

	switch srcact {
	case ValueAdd:
		niceDescShort = "Added property %v"
		niceDesc = "Added property %v"
	case ValueDelete:
		niceDescShort = "Property %v removed"
		niceDesc = "Property %v removed"
	case ValueChange, SequenceChange:
		niceDescShort = "Property %v has changed"
		niceDesc = "Property %v has changed"
	}

	return fmt.Sprintf(niceDescShort, key), fmt.Sprintf(niceDesc, key )
}


func GetNiceTextForPage(srcact ApplyAction, pageName string) (string, string) {

	var niceDesc string = ""
	var niceDescShort string = ""

	switch srcact {
	case ValueAdd:
		niceDescShort = "Page %v was added"
		niceDesc = "Page %v was added"
	case ValueDelete:
		niceDescShort = "Page %v is deleted"
		niceDesc = "Page %v is deleted"
	case ValueChange:
		niceDescShort = "Page %v has changed"
		niceDesc = "Page %v has changed"
	case SequenceChange:
		niceDescShort = "Sequence inside page %v has changed"
		niceDesc = "Sequence inside page %v has changed"
	}

	return fmt.Sprintf(niceDescShort, pageName), fmt.Sprintf(niceDesc, pageName )
}

func GetNiceTextForArtboard(srcact ApplyAction, artboardName string, pageName string) (string, string) {

	var niceDesc = ""
	var niceDescShort = ""

	switch srcact {
	case ValueAdd:
		niceDescShort = "Artboard %v was added"
		niceDesc = "Artboard %v was added to page %v"
	case ValueDelete:
		niceDescShort = "Artboard %v is deleted"
		niceDesc = "Artboard %v is deleted from page %v"
	case ValueChange:
		niceDescShort = "Artboard %v has changed"
		niceDesc = "Artboard %v has changed on page %v"
	case SequenceChange:
		niceDescShort = "Sequence of items inside %v has changed"
		niceDesc = "Sequence of items inside %v has changed on page %v"
	}

	return fmt.Sprintf(niceDescShort, artboardName), fmt.Sprintf(niceDesc, artboardName, pageName )
}

func GetNiceTextForUnknownLayer(srcact ApplyAction, layerName string, layerPath string) (string, string) {

	var niceDesc string = ""
	var niceDescShort string = ""

	switch srcact {
	case ValueAdd:
		niceDescShort = "New layer %v"
		niceDesc = "New layer %v was added at location %v"
	case ValueDelete:
		niceDescShort = "Delete %v layer "
		niceDesc = "Deleted %v layer at location %v"
	case ValueChange:
		niceDescShort = "Layer %v has changed"
		niceDesc = "Layer %v has changed at location %v"
	case SequenceChange:
		niceDescShort = "Layers sequence inside %v has changed"
		niceDesc = "Layers sequence inside %v has changed at location %v"
	}

	return fmt.Sprintf(niceDescShort, layerName), fmt.Sprintf(niceDesc, layerName, layerPath )
}

func GetNiceTextForLayer(srcact ApplyAction, layerName string, pageName string, artboardName string, layerPath string) (string, string) {

	var niceDesc string = ""
	var niceDescShort string = ""

	switch srcact {
	case ValueAdd:
		niceDescShort = "New layer %v"
		niceDesc = "New layer %v was added to page %v in %v artboard (%v)"
	case ValueDelete:
		niceDescShort = "Delete %v layer "
		niceDesc = "Deleted %v layer from page %v in %v artboard (%v)"
	case ValueChange:
		niceDescShort = "Layer %v has changed"
		niceDesc = "Layer %v has changed on page %v in %v artboard (%v)"
	case SequenceChange:
		niceDescShort = "Layers sequence inside %v has changed"
		niceDesc = "Layers sequence inside %v has changed on page %v in %v artboard (%v)"
	}

	return fmt.Sprintf(niceDescShort, layerName), fmt.Sprintf(niceDesc, layerName, pageName, artboardName, layerPath )
}