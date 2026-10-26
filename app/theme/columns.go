package theme

// Two columns, Mine and Agents: the room between them, the tray the Agents column sits on, and
// the content width below which they stack.
const (
	ColumnsGap        float32 = 28 // between Mine and the Agents tray
	TrayRadius        float32 = 14
	TrayInset         float32 = 4  // the Agents tray of Home: its padding and the gap of its cards (v34)
	TrayPad           float32 = 16 // inside the tray, left and right
	ColumnsStackBelow float32 = 640
	ColumnHead        float32 = 40 // a column's header, name and a line under it
	ColumnAvatar      float32 = 20
)
