package dataset

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"
)

type leafDef struct {
	path     string // "Electronics > Audio > Headphones"
	min, max float64
	brands   []string
	nouns    []string
	kind     string // generic, fashion, electronics, book, techbook, grocery
	weight   float64
}

var leaves = []leafDef{
	{"Electronics > Phones & Tablets > Smartphones", 149, 1299, []string{"Novaphone", "Pixelon", "Orbit", "Zentek"}, []string{"Smartphone", "5G Phone", "Phone"}, "electronics", 1.2},
	{"Electronics > Phones & Tablets > Tablets", 129, 1099, []string{"Novaphone", "Zentek", "Lumen"}, []string{"Tablet", "Tab", "Drawing Tablet"}, "electronics", 0.8},
	{"Electronics > Computers > Laptops", 399, 2499, []string{"Zentek", "Corelia", "Lumen", "Vertex"}, []string{"Laptop", "Notebook", "Ultrabook", "Gaming Laptop"}, "electronics", 1.1},
	{"Electronics > Computers > Monitors", 129, 899, []string{"Corelia", "Vertex", "Lumen"}, []string{"27\" Monitor", "4K Monitor", "Curved Monitor", "Portable Monitor"}, "electronics", 0.8},
	{"Electronics > Computers > Keyboards & Mice", 15, 179, []string{"Vertex", "Keyforge", "Corelia"}, []string{"Mechanical Keyboard", "Wireless Mouse", "Keyboard", "Trackpad", "Gaming Mouse"}, "electronics", 1.0},
	{"Electronics > Audio > Headphones", 19, 449, []string{"Sonique", "Beatline", "Auralis"}, []string{"Wireless Headphones", "Earbuds", "Over-Ear Headphones", "Noise-Cancelling Headphones"}, "electronics", 1.3},
	{"Electronics > Audio > Speakers", 29, 599, []string{"Sonique", "Auralis", "Boomfield"}, []string{"Bluetooth Speaker", "Soundbar", "Smart Speaker", "Bookshelf Speakers"}, "electronics", 0.9},
	{"Electronics > Cameras", 249, 2999, []string{"Optiq", "Fotonix", "Lumen"}, []string{"Mirrorless Camera", "Action Camera", "Instant Camera", "Camera Lens"}, "electronics", 0.6},
	{"Electronics > Smart Home", 19, 349, []string{"Homely", "Zentek", "Nestwise"}, []string{"Smart Plug", "Security Camera", "Smart Thermostat", "Video Doorbell", "Smart Bulb Pack"}, "electronics", 1.0},
	{"Home & Kitchen > Kitchen > Cookware", 19, 399, []string{"Ferro", "Casa Bella", "ChefLine"}, []string{"Frying Pan", "Cookware Set", "Dutch Oven", "Saucepan", "Knife Set"}, "generic", 1.0},
	{"Home & Kitchen > Kitchen > Small Appliances", 29, 699, []string{"ChefLine", "Brevo", "Kitchora"}, []string{"Blender", "Espresso Machine", "Air Fryer", "Toaster", "Stand Mixer"}, "electronics", 1.0},
	{"Home & Kitchen > Furniture", 49, 1899, []string{"Nordhem", "Oakline", "Habitat & Co"}, []string{"Office Chair", "Sofa", "Bookshelf", "Standing Desk", "Coffee Table"}, "generic", 0.9},
	{"Home & Kitchen > Bedding", 19, 299, []string{"Nordhem", "Dreamloft", "Linenhaus"}, []string{"Duvet", "Pillow Set", "Sheet Set", "Mattress Topper"}, "generic", 0.7},
	{"Home & Kitchen > Decor", 9, 249, []string{"Habitat & Co", "Lumen Home", "Oakline"}, []string{"Table Lamp", "Wall Art", "Vase", "Candle Set", "Area Rug"}, "generic", 0.8},
	{"Fashion > Men > Men's Clothing", 12, 249, []string{"Urbanthread", "Kestrel", "Fieldmark"}, []string{"T-Shirt", "Jeans", "Jacket", "Hoodie", "Chinos", "Oxford Shirt"}, "fashion", 1.3},
	{"Fashion > Women > Women's Clothing", 12, 299, []string{"Urbanthread", "Maison Vela", "Kestrel"}, []string{"Dress", "Blouse", "Jeans", "Cardigan", "Skirt", "Jacket"}, "fashion", 1.4},
	{"Fashion > Shoes", 29, 259, []string{"Stridex", "Kestrel", "Trailmark"}, []string{"Running Shoes", "Sneakers", "Boots", "Loafers", "Sandals"}, "fashion", 1.1},
	{"Fashion > Bags", 19, 499, []string{"Maison Vela", "Trailmark", "Urbanthread"}, []string{"Backpack", "Tote Bag", "Duffel Bag", "Wallet"}, "fashion", 0.7},
	{"Fashion > Watches", 39, 899, []string{"Chronex", "Zentek", "Fieldmark"}, []string{"Smartwatch", "Chronograph Watch", "Field Watch", "Dive Watch"}, "electronics", 0.6},
	{"Sports & Outdoors > Fitness", 9, 1499, []string{"Peakform", "Ironclad", "Flexa"}, []string{"Yoga Mat", "Dumbbell Set", "Resistance Bands", "Treadmill", "Exercise Bike", "Kettlebell"}, "generic", 1.0},
	{"Sports & Outdoors > Camping", 15, 599, []string{"Trailmark", "Summit Gear", "Peakform"}, []string{"Tent", "Sleeping Bag", "Camping Stove", "Headlamp", "Hiking Backpack"}, "generic", 0.8},
	{"Sports & Outdoors > Cycling", 15, 2499, []string{"Veloce", "Summit Gear", "Trailmark"}, []string{"Road Bike", "Bike Helmet", "Bike Lights", "Mountain Bike", "Cycling Gloves"}, "generic", 0.6},
	{"Beauty > Skincare", 8, 129, []string{"Lumiere", "Dermaco", "Velvet & Vine"}, []string{"Moisturizer", "Vitamin C Serum", "Sunscreen SPF 50", "Gentle Cleanser", "Night Cream"}, "generic", 1.0},
	{"Beauty > Makeup", 6, 79, []string{"Lumiere", "Velvet & Vine", "Rouge Atelier"}, []string{"Lipstick", "Foundation", "Mascara", "Eyeshadow Palette", "Concealer"}, "generic", 0.9},
	{"Beauty > Hair Care", 6, 299, []string{"Dermaco", "Silkwave", "Lumiere"}, []string{"Shampoo", "Conditioner", "Hair Dryer", "Hair Straightener", "Hair Oil"}, "generic", 0.7},
	{"Beauty > Fragrance", 25, 219, []string{"Maison Vela", "Rouge Atelier", "Lumiere"}, []string{"Eau de Parfum", "Eau de Toilette", "Body Mist"}, "generic", 0.5},
	{"Books > Fiction", 8, 35, []string{"Penwick Press", "Harbor House", "Blue Lantern"}, nil, "book", 1.0},
	{"Books > Non-fiction", 10, 45, []string{"Penwick Press", "Harbor House", "Meridian Books"}, nil, "book", 0.8},
	{"Books > Technical", 25, 89, []string{"Meridian Books", "Byteworks Press", "Penwick Press"}, nil, "techbook", 0.6},
	{"Toys & Games > Board Games", 15, 89, []string{"Playwell", "Meeple & Co", "Tinkerbox"}, []string{"Strategy Game", "Party Game", "Card Game", "Cooperative Game"}, "generic", 0.6},
	{"Toys & Games > Building Sets", 10, 249, []string{"Tinkerbox", "Brickhaus", "Playwell"}, []string{"Building Set", "Robot Kit", "Model Kit", "Marble Run"}, "generic", 0.5},
	{"Grocery > Coffee & Tea", 6, 49, []string{"Roastery 9", "Highland Leaf", "Bean & Bloom"}, []string{"Whole Bean Coffee", "Green Tea", "Cold Brew Concentrate", "Espresso Pods", "Chai Blend"}, "grocery", 0.8},
	{"Grocery > Snacks", 3, 29, []string{"Nutsy", "Bean & Bloom", "Crunchwell"}, []string{"Trail Mix", "Protein Bars", "Dark Chocolate", "Granola", "Rice Crackers"}, "grocery", 0.7},
}

var productAdjectives = strings.Fields("Classic Pro Ultra Lite Max Mini Plus Air Prime Eco Sport Studio Essential Signature Urban Smart Compact Deluxe Edge Nova")
var productVariants = []string{"X2", "S5", "Gen 3", "Mk II", "2.0", "Plus", "XL", "One", "V3", "Neo"}
var colors = strings.Fields("black white silver gray navy red blue green beige brown pink gold olive")
var bookAdjectives = strings.Fields("Silent Hidden Last Golden Broken Distant Secret Burning Forgotten Midnight Quiet Wild Crimson Endless Paper Glass")
var bookNouns = strings.Fields("Harbor Garden River Kingdom Letters Orchard Lighthouse Mountain Winter Station Archive Compass Tide Atlas Forest Bridge")
var techTopics = []string{"SQL", "Python", "Data Engineering", "Statistics", "Machine Learning", "Go", "Kubernetes", "Cloud Architecture", "Rust", "JavaScript", "Excel", "Power BI", "Tableau", "Spark", "dbt", "Data Visualization"}
var techSuffixes = []string{"for Beginners", "in Practice", "Cookbook", "Deep Dive", ": The Definitive Guide", "Pocket Reference", "for Analysts", "Patterns"}

type productRec struct {
	id          int
	price, cost float64
	leaf        int
	fashion     bool
	launched    time.Time
	retired     time.Time // zero if still active
	quality     float64
}

type customerRec struct {
	id      int
	signup  time.Time
	country string
	city    int
	spend   float64
	orders  int
}

type orderHead struct {
	cust int // index into customers
	at   time.Time
	nth  int // 0 = customer's first order
}

var couponPct = map[string]float64{
	"WELCOME10": .10, "SPRING15": .15, "SUMMER20": .20, "BF25": .25, "CYBER30": .30, "HOLIDAY20": .20, "LOYAL10": .10, "FREESHIP": 0,
}

func blackFriday(y int) time.Time {
	return nthWeekday(y, time.November, time.Thursday, 4).AddDate(0, 0, 1)
}

var monthSeason = [13]float64{0, .85, .8, .9, .9, .95, .95, 1.05, .95, .95, 1.0, 1.35, 1.45}
var weekdaySeason = [7]float64{1.1, 1.05, 1.0, 1.0, 1.0, .95, 1.05} // Sunday first
var hourWeights = []float64{.6, .35, .2, .15, .15, .25, .5, 1, 1.6, 2, 2.2, 2.6, 3.2, 3.1, 2.7, 2.5, 2.4, 2.6, 3, 3.6, 3.9, 3.6, 2.6, 1.4}

func orderSeason(t time.Time) float64 {
	f := monthSeason[t.Month()] * weekdaySeason[t.Weekday()]
	d := truncDay(t)
	bf := blackFriday(t.Year())
	if !d.Before(bf) && d.Before(bf.AddDate(0, 0, 4)) {
		f *= 2.4
	}
	if t.Month() == time.July && t.Day() >= 15 && t.Day() <= 17 {
		f *= 1.8
	}
	if t.Month() == time.December && t.Day() >= 26 {
		f *= .8
	}
	return f
}

func (g *gen) ecommerce() {
	r := newRNG(g.opt.Seed + 1)
	c := g.counts

	// ---------------------------------------------------------------- categories
	type node struct {
		id, depth int
	}
	nodes := map[string]*node{}
	var order []string
	for _, lf := range leaves {
		parts := strings.Split(lf.path, " > ")
		for i := range parts {
			key := strings.Join(parts[:i+1], " > ")
			if _, ok := nodes[key]; !ok {
				nodes[key] = &node{depth: i}
				order = append(order, key)
			}
		}
	}
	sort.SliceStable(order, func(a, b int) bool { return nodes[order[a]].depth < nodes[order[b]].depth })
	cat := newInserter(g.ctx, g.conn, "categories", "category_id", "name", "parent_category_id")
	for i, key := range order {
		nodes[key].id = i + 1
	}
	for _, key := range order {
		parts := strings.Split(key, " > ")
		var parent any
		if len(parts) > 1 {
			parent = nodes[strings.Join(parts[:len(parts)-1], " > ")].id
		}
		cat.Row(nodes[key].id, parts[len(parts)-1], parent)
	}
	cat.Flush()

	// ---------------------------------------------------------------- suppliers
	supCountries := []string{"China", "China", "China", "Vietnam", "India", "India", "United States", "Germany", "Mexico", "Taiwan", "South Korea", "Italy", "Turkey", "Bangladesh", "Japan"}
	supWords1 := strings.Fields("Pacific Golden Summit Evergreen Blue Harbor Silverline Pioneer Atlas Orient Northstar Sunrise Crescent Delta Apex Lotus Meridian Keystone Horizon Vertex Cobalt")
	supWords2 := strings.Fields("Manufacturing Trading Industries Supply Goods Textiles Electronics Imports Group Works")
	sup := newInserter(g.ctx, g.conn, "suppliers", "supplier_id", "name", "country", "contact_email", "rating", "since")
	usedSup := map[string]bool{}
	for i := 1; i <= c.Suppliers; i++ {
		var name string
		for {
			name = r.pick(supWords1) + " " + r.pick(supWords2)
			if r.chance(.4) {
				name += [...]string{" Co.", " Ltd.", " LLC", " Inc."}[r.IntN(4)]
			}
			if !usedSup[name] {
				usedSup[name] = true
				break
			}
		}
		var email any
		if !r.chance(.12) {
			email = "sales@" + strings.ToLower(strings.NewReplacer(" ", "", ".", "").Replace(strings.Fields(name)[0]+strings.Fields(name)[1])) + ".com"
		}
		sup.Row(i, name, r.pick(supCountries), email, Dec{clamp(r.normal(3.8, .6), 1, 5), 1}, Date(r.randTime(day(2015, 1, 1), day(2023, 6, 1))))
	}
	sup.Flush()

	// ---------------------------------------------------------------- products
	leafW := make([]float64, len(leaves))
	for i, lf := range leaves {
		leafW[i] = lf.weight
	}
	leafPick := newSampler(leafW)
	prod := newInserter(g.ctx, g.conn, "products", "product_id", "sku", "name", "category_id", "supplier_id", "brand", "unit_price", "unit_cost", "attributes", "launched_on", "is_active")
	g.products = make([]productRec, c.Products)
	popularity := make([]float64, c.Products)
	for i := range g.products {
		li := leafPick.draw(r)
		lf := leaves[li]
		id := i + 1
		brand := r.pick(lf.brands)
		var name string
		switch lf.kind {
		case "book":
			switch r.IntN(3) {
			case 0:
				name = "The " + r.pick(bookAdjectives) + " " + r.pick(bookNouns)
			case 1:
				name = r.pick(bookNouns) + " of " + r.pick(bookAdjectives) + " " + r.pick(bookNouns) + "s"
			default:
				name = "A " + r.pick(bookAdjectives) + " " + r.pick(bookNouns)
			}
		case "techbook":
			s := r.pick(techSuffixes)
			name = r.pick(techTopics)
			if strings.HasPrefix(s, ":") {
				name += s
			} else {
				name += " " + s
			}
		default:
			name = brand + " " + r.pick(productAdjectives) + " " + r.pick(lf.nouns)
			if r.chance(.35) {
				name += " " + r.pick(productVariants)
			}
		}
		price := math.Exp(r.between(math.Log(lf.min), math.Log(lf.max)))
		price = math.Floor(price) + .99
		if price > 100 && r.chance(.5) {
			price = math.Round(price/10)*10 - .01
		}
		cost := round2(price * r.between(.35, .72))
		attrs := map[string]any{"color": r.pick(colors)}
		switch lf.kind {
		case "electronics":
			attrs["warranty_years"] = []int{1, 1, 2, 2, 3}[r.IntN(5)]
			attrs["weight_kg"] = round2(r.between(.05, 6))
			if r.chance(.4) {
				attrs["wireless"] = r.chance(.7)
			}
		case "fashion":
			attrs["sizes"] = [][]string{{"S", "M", "L"}, {"XS", "S", "M", "L", "XL"}, {"M", "L", "XL", "XXL"}, {"One Size"}}[r.IntN(4)]
			attrs["material"] = r.pick([]string{"cotton", "polyester", "wool", "leather", "denim", "linen"})
		case "book", "techbook":
			delete(attrs, "color")
			attrs["pages"] = r.intn(120, 820)
			attrs["format"] = r.pick([]string{"paperback", "paperback", "hardcover", "ebook"})
			attrs["language"] = r.pick([]string{"English", "English", "English", "Spanish", "German", "French"})
		case "grocery":
			delete(attrs, "color")
			attrs["weight_kg"] = round2(r.between(.1, 1.5))
			attrs["organic"] = r.chance(.35)
		default:
			attrs["weight_kg"] = round2(r.between(.1, 25))
		}
		var attrJSON any
		if !r.chance(.04) {
			b, _ := json.Marshal(attrs)
			attrJSON = string(b)
		}
		var launched time.Time
		if r.chance(.4) {
			launched = r.randTime(day(2018, 1, 1), day(2021, 1, 1))
		} else {
			launched = r.randTime(day(2021, 1, 1), day(2025, 10, 15))
		}
		launched = truncDay(launched)
		p := productRec{id: id, price: price, cost: cost, leaf: li, fashion: lf.kind == "fashion", launched: launched,
			quality: clamp(r.normal(3.9, .55), 1.8, 4.9)}
		active := true
		if r.chance(.12) {
			ret := launched.AddDate(0, 0, r.intn(365, 1460))
			if ret.Before(day(2025, 12, 1)) {
				p.retired = ret
				active = false
			}
		}
		g.products[i] = p
		popularity[i] = r.lognormal(0, 1.1) / math.Pow(price, .35)
		if lf.kind == "grocery" {
			popularity[i] *= 1.6
		}
		prefix := strings.ToUpper(strings.ReplaceAll(lf.path[:3], " ", "X"))
		prod.Row(id, fmt.Sprintf("%s-%05d", prefix, 10000+id), name, nodes[lf.path].id, 1+r.IntN(c.Suppliers), brand,
			D2(price), D2(cost), attrJSON, Date(launched), active)
	}
	prod.Flush()
	prodPick := newSampler(popularity)

	// ---------------------------------------------------------------- customers (in memory first)
	cityW := make([]float64, len(cities))
	for i, ct := range cities {
		cityW[i] = ct.W
	}
	cityPick := newSampler(cityW)
	g.customers = make([]customerRec, c.Customers)
	signups := make([]time.Time, c.Customers)
	span := g.end.Sub(g.start)
	for i := range signups {
		u := math.Pow(r.Float64(), .72) // more signups later: the business is growing
		signups[i] = g.start.Add(time.Duration(u * float64(span)))
	}
	slices.SortFunc(signups, func(a, b time.Time) int { return a.Compare(b) })
	for i := range g.customers {
		ci := cityPick.draw(r)
		g.customers[i] = customerRec{id: i + 1, signup: signups[i], city: ci, country: cities[ci].Country}
	}

	// order counts: 15% never order, the rest heavy-tailed
	weights := make([]float64, c.Customers)
	sumW := 0.0
	active := 0
	for i := range weights {
		if r.chance(.15) {
			continue
		}
		// customers who joined late have had less time to order
		months := g.end.Sub(g.customers[i].signup).Hours() / 24 / 30
		weights[i] = r.lognormal(0, 1.0) * math.Min(1, (months+1)/14)
		sumW += weights[i]
		active++
	}
	var heads []orderHead
	for i, w := range weights {
		if w == 0 {
			continue
		}
		n := int(math.Max(1, math.Round(w/sumW*float64(c.Orders))))
		cu := &g.customers[i]
		lifeEnd := g.end
		if !r.chance(.3) {
			lifeEnd = minTime(g.end, addDays(cu.signup, r.ExpFloat64()*600+30))
		}
		first := minTime(addDays(cu.signup, r.ExpFloat64()*6), g.end.Add(-time.Hour))
		heads = append(heads, orderHead{cust: i, at: g.orderTime(r, first), nth: 0})
		for k := 1; k < n; k++ {
			var t time.Time
			for try := 0; try < 25; try++ {
				t = r.randTime(first, lifeEnd)
				if r.Float64()*3.7 < orderSeason(t) {
					break
				}
			}
			heads = append(heads, orderHead{cust: i, at: g.orderTime(r, t), nth: k})
		}
	}
	slices.SortStableFunc(heads, func(a, b orderHead) int { return a.at.Compare(b.at) })

	// ---------------------------------------------------------------- orders and children
	ord := newInserter(g.ctx, g.conn, "orders", "order_id", "customer_id", "order_date", "status", "channel", "coupon_code", "shipping_fee")
	itm := newInserter(g.ctx, g.conn, "order_items", "order_item_id", "order_id", "product_id", "quantity", "unit_price", "discount")
	pay := newInserter(g.ctx, g.conn, "payments", "payment_id", "order_id", "payment_method", "amount", "paid_at", "status")
	shp := newInserter(g.ctx, g.conn, "shipments", "shipment_id", "order_id", "carrier", "shipped_at", "delivered_at")
	rev := newInserter(g.ctx, g.conn, "reviews", "review_id", "product_id", "customer_id", "rating", "review_date", "helpful_votes")
	itemCountW := []float64{40, 25, 15, 9, 6, 3, 1.2, .8}
	discounts := []float64{.05, .10, .15, .20, .25, .30}
	discountW := []float64{3, 4, 3, 2, 1, .5}
	reviewed := map[uint64]bool{}
	itemID, payID, shipID, revID := 0, 0, 0, 0
	seenNth := make([]int, c.Customers)

	for oi, h := range heads {
		orderID := oi + 1
		cu := &g.customers[h.cust]
		country := cu.country
		at := h.at
		age := g.end.Sub(at).Hours() / 24

		year := at.Year()
		mobileShare := .25 + .05*float64(year-2021)
		channel := "web"
		switch x := r.Float64(); {
		case x < .1:
			channel = "store"
		case x < .1+mobileShare:
			channel = "mobile"
		}

		// items
		nItems := r.weighted(itemCountW) + 1
		subtotal := 0.0
		hasFashion := false
		inOrder := map[int]bool{}
		type line struct{ prod int }
		var lines []line
		for k := 0; k < nItems; k++ {
			var p *productRec
			for try := 0; try < 12; try++ {
				cand := &g.products[prodPick.draw(r)]
				if cand.launched.After(at) || (!cand.retired.IsZero() && cand.retired.Before(at)) || inOrder[cand.id] {
					continue
				}
				p = cand
				break
			}
			if p == nil {
				continue
			}
			inOrder[p.id] = true
			qty := 1
			if x := r.Float64(); x > .8 {
				qty = [...]int{2, 2, 2, 3, 3, 4, 5}[r.IntN(7)]
			}
			if leaves[p.leaf].kind == "grocery" && r.chance(.5) {
				qty += r.IntN(4)
			}
			unit := round2(p.price * (1 - .03*float64(2025-year)) * r.between(.97, 1.02))
			disc := 0.0
			pDisc := .22
			if at.Month() >= 11 {
				pDisc = .38
			}
			if r.chance(pDisc) {
				disc = discounts[r.weighted(discountW)]
			}
			itemID++
			itm.Row(itemID, orderID, p.id, qty, D2(unit), Dec{disc, 2})
			subtotal += float64(qty) * unit * (1 - disc)
			hasFashion = hasFashion || p.fashion
			lines = append(lines, line{p.id})
		}
		if len(lines) == 0 { // product catalog too young for this date: fall back to the oldest product
			itemID++
			p := &g.products[0]
			for k := range g.products {
				if g.products[k].launched.Before(g.products[p.id-1].launched) {
					p = &g.products[k]
				}
			}
			unit := round2(p.price)
			itm.Row(itemID, orderID, p.id, 1, D2(unit), Dec{0, 2})
			subtotal += unit
			lines = append(lines, line{p.id})
		}
		subtotal = round2(subtotal)

		// coupon
		var coupon any
		couponCode := ""
		bf := blackFriday(year)
		d := truncDay(at)
		switch {
		case h.nth == 0 && r.chance(.35):
			couponCode = "WELCOME10"
		case !d.Before(bf) && d.Before(bf.AddDate(0, 0, 4)) && r.chance(.3):
			couponCode = "BF25"
			if d.Equal(bf.AddDate(0, 0, 3)) {
				couponCode = "CYBER30"
			}
		case at.Month() == time.December && at.Day() <= 24 && r.chance(.15):
			couponCode = "HOLIDAY20"
		case at.Month() >= 3 && at.Month() <= 5 && r.chance(.06):
			couponCode = "SPRING15"
		case at.Month() >= 6 && at.Month() <= 8 && r.chance(.07):
			couponCode = "SUMMER20"
		case r.chance(.04):
			couponCode = []string{"LOYAL10", "FREESHIP"}[r.IntN(2)]
		}
		if couponCode != "" {
			coupon = couponCode
		}
		shipping := 0.0
		if channel != "store" && subtotal < 50 && couponCode != "FREESHIP" {
			shipping = []float64{4.99, 4.99, 7.99}[r.IntN(3)]
		}
		total := round2(subtotal*(1-couponPct[couponCode]) + shipping)

		// status
		status := "delivered"
		switch {
		case channel == "store":
			if r.chance(.03) && age > 12 {
				status = "returned"
			}
		case age < 2:
			status = []string{"pending", "pending", "processing"}[r.IntN(3)]
		case age < 5:
			if r.chance(.2) {
				status = "processing"
			} else {
				status = "shipped"
			}
		case age < 12 && r.chance(.4):
			status = "shipped"
		}
		if age >= 2 && channel != "store" && r.chance(.055) {
			status = "cancelled"
		} else if status == "delivered" && age >= 12 {
			p := .04
			if hasFashion {
				p += .07
			}
			if r.chance(p) {
				status = "returned"
			}
		}
		ord.Row(orderID, cu.id, DateTime(at), status, channel, coupon, D2(shipping))

		// shipment
		var delivered time.Time
		if channel == "store" {
			delivered = at
		} else if status == "shipped" || status == "delivered" || status == "returned" {
			shipped := addDays(at, r.between(.1, 1.2)+r.ExpFloat64()*.8)
			if isWeekend(shipped) {
				shipped = addDays(shipped, 2)
			}
			shipped = minTime(shipped, g.end.Add(-time.Hour))
			carrier := carrierFor(r, country)
			var deliv any
			if status != "shipped" {
				transit := 1 + r.ExpFloat64()*2.2
				if country != "United States" && country != "United Kingdom" && country != "Germany" {
					transit += 1.5
				}
				delivered = minTime(addDays(shipped, transit), g.end.Add(-time.Minute))
				deliv = DateTime(delivered)
			}
			shipID++
			shp.Row(shipID, orderID, carrier, DateTime(shipped), deliv)
		}

		// payments
		method := paymentMethod(r, country)
		payAt := at.Add(time.Duration(r.IntN(300)+5) * time.Second)
		if method == "cash_on_delivery" && !delivered.IsZero() {
			payAt = delivered
		}
		netPaid := 0.0
		switch status {
		case "pending":
			if r.chance(.5) {
				payID++
				pay.Row(payID, orderID, method, D2(total), DateTime(payAt), "paid")
				netPaid = total
			}
		case "cancelled":
			payID++
			if r.chance(.4) {
				pay.Row(payID, orderID, method, D2(total), DateTime(payAt), "failed")
			} else {
				pay.Row(payID, orderID, method, D2(total), DateTime(payAt), "paid")
				payID++
				pay.Row(payID, orderID, method, D2(total), DateTime(minTime(addDays(payAt, r.between(.2, 4)), g.end)), "refunded")
			}
		default:
			if method == "cash_on_delivery" && delivered.IsZero() {
				break // collected on delivery
			}
			if r.chance(.04) {
				payID++
				pay.Row(payID, orderID, "card", D2(total), DateTime(payAt.Add(-2*time.Minute)), "failed")
			}
			if total > 40 && r.chance(.03) {
				gc := round2(math.Min(total-5, float64(r.intn(2, 10))*5))
				payID++
				pay.Row(payID, orderID, "gift_card", D2(gc), DateTime(payAt), "paid")
				payID++
				pay.Row(payID, orderID, "card", D2(total-gc), DateTime(payAt.Add(time.Second)), "paid")
			} else {
				payID++
				pay.Row(payID, orderID, method, D2(total), DateTime(payAt), "paid")
			}
			netPaid = total
			if status == "returned" {
				refundAt := minTime(addDays(maxTime(delivered, at), r.between(8, 30)), g.end)
				payID++
				pay.Row(payID, orderID, method, D2(total), DateTime(refundAt), "refunded")
				netPaid = 0
			}
		}
		cu.spend += netPaid
		cu.orders++
		seenNth[h.cust]++

		// reviews
		if !delivered.IsZero() && (status == "delivered" || status == "returned") {
			for _, ln := range lines {
				p := &g.products[ln.prod-1]
				pr := .12
				bias := 0.0
				if status == "returned" {
					pr, bias = .3, -1.6
				}
				key := uint64(cu.id)<<32 | uint64(p.id)
				if reviewed[key] || !r.chance(pr) {
					continue
				}
				revDate := addDays(delivered, 1+r.ExpFloat64()*9)
				if revDate.After(g.end) {
					continue
				}
				reviewed[key] = true
				rating := int(math.Round(clamp(r.normal(p.quality+bias, .9), 1, 5)))
				votes := 0
				if r.chance(.45) {
					votes = int(r.ExpFloat64() * 4)
					if r.chance(.03) {
						votes += r.intn(20, 300)
					}
				}
				revID++
				rev.Row(revID, p.id, cu.id, rating, Date(revDate), votes)
			}
		}
	}
	ord.Flush()
	itm.Flush()
	pay.Flush()
	shp.Flush()
	rev.Flush()

	// ---------------------------------------------------------------- write customers
	spendRank := make([]int, len(g.customers))
	for i := range spendRank {
		spendRank[i] = i
	}
	slices.SortFunc(spendRank, func(a, b int) int {
		if g.customers[a].spend == g.customers[b].spend {
			return a - b
		}
		if g.customers[a].spend > g.customers[b].spend {
			return -1
		}
		return 1
	})
	tier := make([]string, len(g.customers))
	for rank, idx := range spendRank {
		q := float64(rank) / float64(len(spendRank))
		switch {
		case g.customers[idx].spend == 0:
			tier[idx] = "bronze"
		case q < .02:
			tier[idx] = "platinum"
		case q < .10:
			tier[idx] = "gold"
		case q < .30:
			tier[idx] = "silver"
		default:
			tier[idx] = "bronze"
		}
	}
	channels := []string{"organic_search", "paid_search", "social", "referral", "email", "affiliate", "direct"}
	channelW := []float64{30, 20, 18, 12, 8, 7, 5}
	cust := newInserter(g.ctx, g.conn, "customers", "customer_id", "first_name", "last_name", "email", "phone", "gender",
		"birth_date", "city", "state", "country", "signup_at", "loyalty_tier", "acquisition_channel", "referred_by_customer_id")
	usedEmail := map[string]bool{}
	for i := range g.customers {
		cu := &g.customers[i]
		var gender any
		first := ""
		switch x := r.Float64(); {
		case x < .48:
			gender, first = "F", r.pick(femaleNames)
		case x < .95:
			gender, first = "M", r.pick(maleNames)
		default:
			if r.chance(.5) {
				first = r.pick(femaleNames)
			} else {
				first = r.pick(maleNames)
			}
		}
		last := r.pick(lastNames)
		var email string
		for {
			email = emailLocal(first, last)
			switch r.IntN(4) {
			case 0:
				email += fmt.Sprint(r.intn(1, 99))
			case 1:
				email += fmt.Sprint(r.intn(1960, 2005))
			case 2:
				email = strings.ToLower(asciiFold.Replace(first))[:1] + strings.ToLower(asciiFold.Replace(last)) + fmt.Sprint(r.intn(1, 999))
			}
			email += "@" + r.pick(emailDomains)
			if !usedEmail[email] {
				usedEmail[email] = true
				break
			}
		}
		if r.chance(.03) { // data-quality quirk: mixed case emails
			email = strings.ToUpper(email[:1]) + email[1:]
		}
		ct := cities[cu.city]
		var phone any
		if !r.chance(.3) {
			phone = phoneNumber(r, cu.country)
		}
		var birth any
		if !r.chance(.08) {
			birth = Date(truncDay(cu.signup.AddDate(-r.intn(18, 74), 0, -r.IntN(365))))
		}
		cityName := ct.Name
		if r.chance(.005) {
			cityName = strings.ToLower(cityName)
		} else if r.chance(.005) {
			cityName += " "
		}
		var state any
		if ct.State != "" {
			state = ct.State
		}
		ch := channels[r.weighted(channelW)]
		var ref any
		if ch == "referral" && i > 10 {
			ref = 1 + r.IntN(i-1)
		}
		cust.Row(cu.id, first, last, email, phone, gender, birth, cityName, state, cu.country,
			DateTime(cu.signup), tier[i], ch, ref)
	}
	cust.Flush()
	g.logf("e-commerce: %d customers, %d products, %d orders, %d items, %d payments, %d shipments, %d reviews",
		cust.Total, prod.Total, ord.Total, itm.Total, pay.Total, shp.Total, rev.Total)
}

// orderTime keeps the date of t but redraws the time of day from realistic shopping hours.
func (g *gen) orderTime(r rng, t time.Time) time.Time {
	d := truncDay(t)
	h := r.weighted(hourWeights)
	out := d.Add(time.Duration(h)*time.Hour + time.Duration(r.IntN(3600))*time.Second)
	if out.After(g.end) {
		out = g.end.Add(-time.Duration(r.IntN(7200)+60) * time.Second)
	}
	return out
}

func carrierFor(r rng, country string) string {
	switch country {
	case "United States":
		return r.pick([]string{"UPS", "UPS", "FedEx", "USPS", "USPS"})
	case "India":
		return r.pick([]string{"BlueDart", "Delhivery", "Delhivery", "DHL"})
	case "United Kingdom":
		return r.pick([]string{"Royal Mail", "Royal Mail", "DPD", "DHL"})
	case "Germany", "Netherlands":
		return r.pick([]string{"DHL", "DHL", "Hermes", "DPD"})
	case "Canada":
		return r.pick([]string{"Canada Post", "UPS", "FedEx"})
	}
	return r.pick([]string{"DHL", "FedEx", "UPS"})
}

func paymentMethod(r rng, country string) string {
	w := []float64{58, 20, 6, 4, 3} // card, paypal, bank_transfer, gift_card, cod
	switch country {
	case "India":
		w[4] = 35
	case "Brazil", "Mexico":
		w[4] = 20
	case "Germany", "Netherlands":
		w[2] = 18
	}
	return []string{"card", "paypal", "bank_transfer", "gift_card", "cash_on_delivery"}[r.weighted(w)]
}

func phoneNumber(r rng, country string) string {
	p := phonePrefix[country]
	if country == "United States" || country == "Canada" {
		a, b, c := r.intn(201, 989), r.intn(200, 999), r.intn(0, 9999)
		switch r.IntN(3) {
		case 0:
			return fmt.Sprintf("(%d) %d-%04d", a, b, c)
		case 1:
			return fmt.Sprintf("%d-%d-%04d", a, b, c)
		default:
			return fmt.Sprintf("+1 %d %d %04d", a, b, c)
		}
	}
	return fmt.Sprintf("%s %d %d", p, r.intn(10, 99), r.intn(1000000, 99999999))
}
