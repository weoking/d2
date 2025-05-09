package d2sequence

import (
	"context"
	"testing"

	"oss.terrastruct.com/d2/d2graph"
	"oss.terrastruct.com/d2/d2target"
	"oss.terrastruct.com/d2/lib/geo"
	"oss.terrastruct.com/d2/lib/label"
	"oss.terrastruct.com/d2/lib/log"
	"oss.terrastruct.com/d2/lib/shape"
)

func TestBasicSequenceDiagram(t *testing.T) {
	// ┌────────┐              ┌────────┐
	// │   n1   │              │   n2   │
	// └────┬───┘              └────┬───┘
	//      │                       │
	//      ├───────────────────────►
	//      │                       │
	//      ◄───────────────────────┤
	//      │                       │
	//      ├───────────────────────►
	//      │                       │
	//      ◄───────────────────────┤
	//      │                       │
	g := d2graph.NewGraph(nil)
	g.Root.Attributes.Shape = d2graph.Scalar{Value: d2target.ShapeSequenceDiagram}
	n1 := g.Root.EnsureChild([]string{"n1"})
	n1.Box = geo.NewBox(nil, 100, 100)
	n2 := g.Root.EnsureChild([]string{"n2"})
	n2.Box = geo.NewBox(nil, 30, 30)

	g.Edges = []*d2graph.Edge{
		{
			Src:   n1,
			Dst:   n2,
			Index: 0,
			Attributes: d2graph.Attributes{
				Label: d2graph.Scalar{Value: "left to right"},
			},
		},
		{
			Src:   n2,
			Dst:   n1,
			Index: 0,
			Attributes: d2graph.Attributes{
				Label: d2graph.Scalar{Value: "right to left"},
			},
		},
		{
			Src:   n1,
			Dst:   n2,
			Index: 1,
		},
		{
			Src:   n2,
			Dst:   n1,
			Index: 1,
		},
	}
	nEdges := len(g.Edges)

	ctx := log.WithTB(context.Background(), t, nil)
	Layout(ctx, g, func(ctx context.Context, g *d2graph.Graph) error {
		// just set some position as if it had been properly placed
		for _, obj := range g.Objects {
			obj.TopLeft = geo.NewPoint(0, 0)
		}

		for _, edge := range g.Edges {
			edge.Route = []*geo.Point{geo.NewPoint(1, 1)}
		}
		return nil
	})

	// asserts that actors were placed in the expected x order and at y=0
	actors := []*d2graph.Object{
		g.Objects[0],
		g.Objects[1],
	}
	for i := 1; i < len(actors); i++ {
		if actors[i].TopLeft.X < actors[i-1].TopLeft.X {
			t.Fatalf("expected actor[%d].TopLeft.X > actor[%d].TopLeft.X", i, i-1)
		}
		actorBottom := actors[i].TopLeft.Y + actors[i].Height
		prevActorBottom := actors[i-1].TopLeft.Y + actors[i-1].Height
		if actorBottom != prevActorBottom {
			t.Fatalf("expected actor[%d] and actor[%d] to be at the same bottom y", i, i-1)
		}
	}

	nExpectedEdges := nEdges + len(actors)
	if len(g.Edges) != nExpectedEdges {
		t.Fatalf("expected %d edges, got %d", nExpectedEdges, len(g.Edges))
	}

	// assert that edges were placed in y order and have the endpoints at their actors
	// uses `nEdges` because Layout creates some vertical edges to represent the actor lifeline
	for i := 0; i < nEdges; i++ {
		edge := g.Edges[i]
		if len(edge.Route) != 2 {
			t.Fatalf("expected edge[%d] to have only 2 points", i)
		}
		if edge.Route[0].Y != edge.Route[1].Y {
			t.Fatalf("expected edge[%d] to be a horizontal line", i)
		}
		if edge.Src.TopLeft.X < edge.Dst.TopLeft.X {
			// left to right
			if edge.Route[0].X != edge.Src.Center().X {
				t.Fatalf("expected edge[%d] x to be at the actor center", i)
			}

			if edge.Route[1].X != edge.Dst.Center().X {
				t.Fatalf("expected edge[%d] x to be at the actor center", i)
			}
		} else {
			if edge.Route[0].X != edge.Src.Center().X {
				t.Fatalf("expected edge[%d] x to be at the actor center", i)
			}

			if edge.Route[1].X != edge.Dst.Center().X {
				t.Fatalf("expected edge[%d] x to be at the actor center", i)
			}
		}
		if i > 0 {
			prevEdge := g.Edges[i-1]
			if edge.Route[0].Y < prevEdge.Route[0].Y {
				t.Fatalf("expected edge[%d].TopLeft.Y > edge[%d].TopLeft.Y", i, i-1)
			}
		}
	}

	lastSequenceEdge := g.Edges[nEdges-1]
	for i := nEdges; i < nExpectedEdges; i++ {
		edge := g.Edges[i]
		if len(edge.Route) != 2 {
			t.Fatalf("expected lifeline edge[%d] to have only 2 points", i)
		}
		if edge.Route[0].X != edge.Route[1].X {
			t.Fatalf("expected lifeline edge[%d] to be a vertical line", i)
		}
		if edge.Route[0].X != edge.Src.Center().X {
			t.Fatalf("expected lifeline edge[%d] x to be at the actor center", i)
		}
		if edge.Route[0].Y != edge.Src.Height+edge.Src.TopLeft.Y {
			t.Fatalf("expected lifeline edge[%d] to start at the bottom of the source actor", i)
		}
		if edge.Route[1].Y < lastSequenceEdge.Route[0].Y {
			t.Fatalf("expected lifeline edge[%d] to end after the last sequence edge", i)
		}
	}

	// check label positions
	if *g.Edges[0].LabelPosition != string(label.OutsideTopCenter) {
		t.Fatalf("expected edge label to be placed on %s, got %s", string(label.OutsideTopCenter), *g.Edges[0].LabelPosition)
	}

	if *g.Edges[1].LabelPosition != string(label.OutsideBottomCenter) {
		t.Fatalf("expected edge label to be placed on %s, got %s", string(label.OutsideBottomCenter), *g.Edges[0].LabelPosition)
	}
}

func TestSpansSequenceDiagram(t *testing.T) {
	//   ┌─────┐                 ┌─────┐
	//   │  a  │                 │  b  │
	//   └──┬──┘                 └──┬──┘
	//      ├┐────────────────────►┌┤
	//   t1 ││                     ││ t1
	//      ├┘◄────────────────────└┤
	//      ├┐──────────────────────►
	//   t2 ││                      │
	//      ├┘◄─────────────────────┤
	g := d2graph.NewGraph(nil)
	g.Root.Attributes.Shape = d2graph.Scalar{Value: d2target.ShapeSequenceDiagram}
	a := g.Root.EnsureChild([]string{"a"})
	a.Box = geo.NewBox(nil, 100, 100)
	a.Attributes = d2graph.Attributes{
		Shape: d2graph.Scalar{Value: shape.PERSON_TYPE},
	}
	a_t1 := a.EnsureChild([]string{"t1"})
	a_t1.Attributes = d2graph.Attributes{
		Shape: d2graph.Scalar{Value: shape.DIAMOND_TYPE},
		Label: d2graph.Scalar{Value: "label"},
	}
	a_t2 := a.EnsureChild([]string{"t2"})
	b := g.Root.EnsureChild([]string{"b"})
	b.Box = geo.NewBox(nil, 30, 30)
	b_t1 := b.EnsureChild([]string{"t1"})

	g.Edges = []*d2graph.Edge{
		{
			Src:   a_t1,
			Dst:   b_t1,
			Index: 0,
		}, {
			Src:   b_t1,
			Dst:   a_t1,
			Index: 0,
		}, {
			Src:   a_t2,
			Dst:   b,
			Index: 0,
		}, {
			Src:   b,
			Dst:   a_t2,
			Index: 0,
		},
	}

	ctx := log.WithTB(context.Background(), t, nil)
	Layout(ctx, g, func(ctx context.Context, g *d2graph.Graph) error {
		// just set some position as if it had been properly placed
		for _, obj := range g.Objects {
			obj.TopLeft = geo.NewPoint(0, 0)
		}

		for _, edge := range g.Edges {
			edge.Route = []*geo.Point{geo.NewPoint(1, 1)}
		}
		return nil
	})

	// check properties
	if a.Attributes.Shape.Value != shape.PERSON_TYPE {
		t.Fatal("actor a shape changed")
	}

	if a_t1.Attributes.Label.Value != "" {
		t.Fatalf("expected no label for span, got %s", a_t1.Attributes.Label.Value)
	}

	if a_t1.Attributes.Shape.Value != shape.SQUARE_TYPE {
		t.Fatalf("expected square shape for span, got %s", a_t1.Attributes.Shape.Value)
	}

	if a_t1.Height != b_t1.Height {
		t.Fatalf("expected a.t1 and b.t1 to have the same height, got %.5f and %.5f", a_t1.Height, b_t1.Height)
	}

	for _, span := range []*d2graph.Object{a_t1, a_t2, b_t1} {
		if span.ZIndex != 1 {
			t.Fatalf("expected span ZIndex=1, got %d", span.ZIndex)
		}
	}

	// Y diff of the 2 first edges
	expectedHeight := g.Edges[1].Route[0].Y - g.Edges[0].Route[0].Y + (2 * SPAN_MESSAGE_PAD)
	if a_t1.Height != expectedHeight {
		t.Fatalf("expected a.t1 height to be %.5f, got %.5f", expectedHeight, a_t1.Height)
	}

	if a_t1.Width != SPAN_BASE_WIDTH {
		t.Fatalf("expected span width to be %.5f, got %.5f", SPAN_BASE_WIDTH, a_t1.Width)
	}

	// check positions
	if a.Center().X != a_t1.Center().X {
		t.Fatal("expected a_t1.X = a.X")
	}
	if a.Center().X != a_t2.Center().X {
		t.Fatal("expected a_t2.X = a.X")
	}
	if b.Center().X != b_t1.Center().X {
		t.Fatal("expected b_t1.X = b.X")
	}
	if a_t1.TopLeft.Y != b_t1.TopLeft.Y {
		t.Fatal("expected a.t1 and b.t1 to be placed at the same Y")
	}
	if a_t1.TopLeft.Y+SPAN_MESSAGE_PAD != g.Edges[0].Route[0].Y {
		t.Fatal("expected a.t1 to be placed at the same Y of the first message")
	}

	// check routes
	if g.Edges[0].Route[0].X != a_t1.TopLeft.X+a_t1.Width {
		t.Fatal("expected the first message to start on a.t1 top right X")
	}

	if g.Edges[0].Route[1].X != b_t1.TopLeft.X {
		t.Fatal("expected the first message to end on b.t1 top left X")
	}

	if g.Edges[2].Route[1].X != b.Center().X {
		t.Fatal("expected the third message to end on b.t1 center X")
	}
}

func TestNestedSequenceDiagrams(t *testing.T) {
	// ┌────────────────────────────────────────┐
	// |     ┌─────┐    container    ┌─────┐    |
	// |     │  a  │                 │  b  │    |            ┌─────┐
	// |     └──┬──┘                 └──┬──┘    ├────edge1───┤  c  │
	// |        ├┐───────sdEdge1──────►┌┤       |            └─────┘
	// |     t1 ││                     ││ t1    |
	// |        ├┘◄──────sdEdge2───────└┤       |
	// └────────────────────────────────────────┘
	g := d2graph.NewGraph(nil)
	container := g.Root.EnsureChild([]string{"container"})
	container.Attributes.Shape = d2graph.Scalar{Value: d2target.ShapeSequenceDiagram}
	container.Box = geo.NewBox(nil, 500, 500)
	a := container.EnsureChild([]string{"a"})
	a.Box = geo.NewBox(nil, 100, 100)
	a.Attributes.Shape = d2graph.Scalar{Value: shape.PERSON_TYPE}
	a_t1 := a.EnsureChild([]string{"t1"})
	b := container.EnsureChild([]string{"b"})
	b.Box = geo.NewBox(nil, 30, 30)
	b_t1 := b.EnsureChild([]string{"t1"})

	c := g.Root.EnsureChild([]string{"c"})
	c.Box = geo.NewBox(nil, 100, 100)
	c.Attributes.Shape = d2graph.Scalar{Value: d2target.ShapeSquare}

	sdEdge1, err := g.Root.Connect(a_t1.AbsIDArray(), b_t1.AbsIDArray(), false, true, "sequence diagram edge 1")
	if err != nil {
		t.Fatal(err)
	}
	sdEdge2, err := g.Root.Connect(b_t1.AbsIDArray(), a_t1.AbsIDArray(), false, true, "sequence diagram edge 2")
	if err != nil {
		t.Fatal(err)
	}

	edge1, err := g.Root.Connect(container.AbsIDArray(), c.AbsIDArray(), false, false, "edge 1")
	if err != nil {
		t.Fatal(err)
	}

	layoutFn := func(ctx context.Context, g *d2graph.Graph) error {
		if len(g.Objects) != 2 {
			t.Fatal("expected only diagram objects for layout")
		}
		for _, obj := range g.Objects {
			if obj == a || obj == a_t1 || obj == b || obj == b_t1 {
				t.Fatal("expected to have removed all sequence diagram objects")
			}
		}
		if len(container.ChildrenArray) != 0 {
			t.Fatalf("expected no `container` children, got %d", len(container.ChildrenArray))
		}

		if len(container.Children) != len(container.ChildrenArray) {
			t.Fatal("container children mismatch")
		}

		for _, edge := range g.Edges {
			if edge == sdEdge1 || edge == sdEdge2 {
				t.Fatal("expected to have removed all sequence diagram edges from graph")
			}
		}
		if g.Edges[0] != edge1 {
			t.Fatal("expected graph edge to be in the graph")
		}

		// just set some position as if it had been properly placed
		for _, obj := range g.Objects {
			obj.TopLeft = geo.NewPoint(0, 0)
		}

		for _, edge := range g.Edges {
			edge.Route = []*geo.Point{geo.NewPoint(1, 1)}
		}

		return nil
	}

	ctx := log.WithTB(context.Background(), t, nil)
	if err = Layout(ctx, g, layoutFn); err != nil {
		t.Fatal(err)
	}

	if len(g.Edges) != 5 {
		t.Fatal("expected graph to have all edges and lifelines after layout")
	}

	for _, obj := range g.Objects {
		if obj.TopLeft == nil {
			t.Fatal("expected to have placed all objects")
		}
	}
	for _, edge := range g.Edges {
		if len(edge.Route) == 0 {
			t.Fatal("expected to have routed all edges")
		}
	}
}
