package main

import "fmt"

//Applies the throwable
func ApplyThrowable(g *Game, item *Item) {

	t := item.GetItemProperty(THROWABLE_NAME).(Throwable)
	fmt.Println("Throwing ", t)
	fmt.Println("Throwing ", t)

	p := GetTilePositions(t.shape)
	fmt.Println("Printing shape positions ", p)

	for _, c := range g.World.Query(g.WorldTags["monsters"]) {

		curPos := c.Components[position].(*Position)
		fmt.Println("Printing Creature pos: ", curPos)
	}

}
