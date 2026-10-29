package dataset

import (
	"math"
	"slices"
	"strings"
	"time"
)

type sectorDef struct {
	name       string
	industries []string
	words      []string
	drift2022  float64 // sector tilt during the 2022 bear market
}

var sectors = []sectorDef{
	{"Technology", []string{"Software", "Semiconductors", "IT Services", "Hardware"}, []string{"Systems", "Software", "Labs", "Technologies", "Micro"}, -.0009},
	{"Healthcare", []string{"Biotechnology", "Pharmaceuticals", "Medical Devices", "Health Insurance"}, []string{"Therapeutics", "Pharma", "Health", "Biosciences", "Medical"}, .0001},
	{"Financials", []string{"Banks", "Insurance", "Asset Management", "Payments"}, []string{"Bancorp", "Financial", "Capital", "Insurance Group", "Pay"}, -.0001},
	{"Energy", []string{"Oil & Gas", "Renewable Energy"}, []string{"Energy", "Petroleum", "Power", "Solar"}, .0016},
	{"Consumer Discretionary", []string{"Retail", "Automobiles", "Hotels & Leisure", "E-commerce"}, []string{"Motors", "Retail", "Resorts", "Brands", "Outfitters"}, -.0007},
	{"Consumer Staples", []string{"Food & Beverage", "Household Products"}, []string{"Foods", "Beverages", "Consumer Goods", "Farms"}, .0002},
	{"Industrials", []string{"Aerospace & Defense", "Machinery", "Logistics", "Airlines"}, []string{"Industries", "Aerospace", "Logistics", "Airways", "Dynamics"}, -.0002},
	{"Utilities", []string{"Electric Utilities", "Water Utilities"}, []string{"Utilities", "Electric", "Water Co."}, .0002},
	{"Materials", []string{"Chemicals", "Metals & Mining"}, []string{"Chemicals", "Mining", "Materials", "Steel"}, 0},
	{"Real Estate", []string{"REITs"}, []string{"Realty", "Properties", "REIT"}, -.0005},
	{"Communication Services", []string{"Media", "Telecom", "Video Games"}, []string{"Media", "Telecom", "Entertainment", "Interactive"}, -.0008},
}

var namePrefixes = strings.Fields("Nova Quanta Helio Apex Terra Lumen Vireo Corva Strato Orbis Kinetic Solara Medix Aero Hydra Velo Omni Zenith Cyra Agri Petra Urbana Bluewater Ironwood Silverpeak Pacifica Atlas Summit Meridian Polaris Cascade Granite Redwood Harbor Evergreen Northgate Crestline Brightway Falcon Aurora Titan Vector Keystone Beacon Pinnacle Horizon Sterling Cobalt Ember Prism")

type marketRegime struct {
	from       time.Time
	drift, vol float64
}

var regimes = []marketRegime{
	{day(2018, 1, 1), .00045, .008},
	{day(2018, 10, 1), -.0006, .013},
	{day(2019, 1, 1), .0008, .008},
	{day(2020, 2, 20), -.012, .038},
	{day(2020, 3, 24), .0016, .016},
	{day(2021, 1, 1), .0008, .009},
	{day(2022, 1, 1), -.0008, .015},
	{day(2023, 1, 1), .0008, .009},
	{day(2025, 4, 1), -.003, .022},
	{day(2025, 4, 15), .0012, .01},
}

func regimeAt(t time.Time) marketRegime {
	cur := regimes[0]
	for _, rg := range regimes {
		if !t.Before(rg.from) {
			cur = rg
		}
	}
	return cur
}

type merchantCat struct {
	name      string
	merchants []string
	min, max  float64 // USD
	w         float64
}

var merchantCats = []merchantCat{
	{"groceries", []string{"FreshMart", "Green Basket", "Daily Grocer", "Corner Market", "Whole Harvest"}, 15, 180, 26},
	{"restaurants", []string{"Bella Napoli", "Sushi Zen", "Taco Loco", "The Burger Joint", "Spice Route", "Café Lumière"}, 12, 120, 18},
	{"coffee", []string{"Bean There", "Daily Grind", "Brew Lab"}, 3, 12, 14},
	{"transport", []string{"QuickRide", "MetroCard", "City Cabs", "FuelUp Gas"}, 5, 70, 12},
	{"shopping", []string{"ShopSphere", "ShopSphere", "MegaMart", "StyleHub", "HomeNest"}, 10, 400, 12},
	{"entertainment", []string{"CinemaMax", "GameVault", "Ticket Hub"}, 8, 90, 5},
	{"travel", []string{"SkyHigh Airlines", "StayWell Hotels", "RoamBnB"}, 80, 1500, 2},
	{"health", []string{"CarePlus Pharmacy", "City Dental", "Vision Clinic"}, 10, 250, 4},
	{"utilities", []string{"PowerGrid Energy", "AquaWorks", "FiberNet Internet", "MobileOne"}, 30, 200, 7},
}

type subscription struct {
	merchant, category string
	usd                float64
}

var subscriptions = []subscription{
	{"StreamFlix", "entertainment", 15.49}, {"SoundWave Music", "entertainment", 10.99},
	{"FitLife Gym", "health", 39.00}, {"CloudBox Storage", "utilities", 2.99}, {"NewsDaily", "entertainment", 8.00},
}

type txnRow struct {
	account            int
	at                 time.Time
	amount             float64
	typ                string
	merchant, category string
}

func (g *gen) finance() {
	r := newRNG(g.opt.Seed + 3)
	c := g.counts
	end := truncDay(g.end)

	// ---------------------------------------------------------------- stocks
	priceStart := day(2018, 1, 2)
	days := tradingDays(priceStart, end)
	dayIdx := map[time.Time]int{}
	for i, d := range days {
		dayIdx[d] = i
	}
	type stockRec struct {
		sector                int
		ipo, delist           int // indexes into days; delist = -1 if still listed
		beta, vol, drift, px0 float64
	}
	stocks := make([]stockRec, c.Stocks)
	st := newInserter(g.ctx, g.conn, "stocks", "stock_id", "ticker", "company_name", "sector", "industry", "exchange", "ipo_date", "delisted_on", "shares_outstanding")
	usedTicker := map[string]bool{}
	usedName := map[string]bool{}
	delisted := map[int]bool{}
	for len(delisted) < 3 {
		delisted[r.IntN(c.Stocks)] = true
	}
	for i := range stocks {
		si := i % len(sectors)
		sec := sectors[si]
		var name string
		for {
			name = r.pick(namePrefixes) + " " + r.pick(sec.words)
			if !usedName[name] {
				usedName[name] = true
				break
			}
		}
		var ticker string
		letters := strings.ToUpper(strings.ReplaceAll(asciiFold.Replace(name), ".", ""))
		for try := 0; ; try++ {
			n := 3 + r.IntN(2)
			b := []byte{letters[0]}
			for len(b) < n {
				ch := letters[1+r.IntN(len(letters)-1)]
				if ch >= 'A' && ch <= 'Z' {
					b = append(b, ch)
				}
			}
			ticker = string(b)
			if !usedTicker[ticker] {
				usedTicker[ticker] = true
				break
			}
		}
		s := stockRec{sector: si, delist: -1,
			beta:  r.between(.6, 1.6),
			vol:   r.between(.008, .025),
			drift: r.normal(.0002, .00035),
			px0:   math.Exp(r.between(math.Log(6), math.Log(380))),
		}
		if sec.name == "Technology" || sec.name == "Communication Services" {
			s.beta += .2
			s.vol += .004
		}
		if r.chance(.25) {
			s.ipo = r.IntN(len(days) - 260)
		}
		if delisted[i] {
			s.ipo = r.IntN(300)
			s.delist = s.ipo + 300 + r.IntN(len(days)-s.ipo-400)
			s.drift = -.0025
		}
		stocks[i] = s
		exch := "NYSE"
		switch x := r.Float64(); {
		case sec.name == "Technology" && x < .75, x < .4:
			exch = "NASDAQ"
		case x > .95:
			exch = "AMEX"
		}
		var del any
		if s.delist >= 0 {
			del = Date(days[s.delist])
		}
		st.Row(i+1, ticker, name, sec.name, r.pick(sec.industries), exch, Date(days[s.ipo]), del, uint64(r.lognormal(math.Log(4e8), .9)))
	}
	st.Flush()

	closes := make([][]float64, c.Stocks) // closes[stock][day], 0 when not trading
	sp := newInserter(g.ctx, g.conn, "stock_prices", "stock_id", "trade_date", "open", "high", "low", "close", "volume")
	market := make([]float64, len(days))
	sectorRet := make([][]float64, len(sectors))
	for si := range sectors {
		sectorRet[si] = make([]float64, len(days))
	}
	for di, d := range days {
		rg := regimeAt(d)
		market[di] = r.normal(rg.drift, rg.vol)
		for si, sec := range sectors {
			tilt := 0.0
			if d.Year() == 2022 {
				tilt = sec.drift2022
			}
			sectorRet[si][di] = r.normal(tilt, .005)
		}
	}
	for i, s := range stocks {
		closes[i] = make([]float64, len(days))
		px := s.px0
		baseVol := r.lognormal(math.Log(2.5e6), .8)
		last := len(days) - 1
		if s.delist >= 0 {
			last = s.delist
		}
		for di := s.ipo; di <= last; di++ {
			ret := s.beta*market[di] + sectorRet[s.sector][di] + r.normal(s.drift, s.vol)
			if di == s.ipo {
				ret = r.normal(.08, .12) // IPO day pop
			}
			open := px * math.Exp(r.normal(0, .004))
			px = math.Max(.5, px*math.Exp(ret))
			hi := math.Max(open, px) * (1 + math.Abs(r.normal(0, .007)))
			lo := math.Min(open, px) * (1 - math.Abs(r.normal(0, .007)))
			vol := baseVol * math.Exp(r.normal(0, .35)) * (1 + 12*math.Abs(ret))
			if di == s.ipo {
				vol *= 6
			}
			closes[i][di] = px
			sp.Row(i+1, Date(days[di]), Dec{open, 4}, Dec{hi, 4}, Dec{lo, 4}, Dec{px, 4}, uint64(math.Min(vol, 4e9)))
		}
	}
	sp.Flush()

	// ---------------------------------------------------------------- exchange rates
	fx := newInserter(g.ctx, g.conn, "exchange_rates", "currency", "rate_date", "rate_to_usd")
	fxVol := map[string]float64{"BRL": .008, "INR": .0025, "CNY": .0015, "MXN": .006, "JPY": .005}
	for _, cur := range []string{"AUD", "BRL", "CAD", "EUR", "GBP", "INR", "JPY", "MXN", "SGD"} {
		base := math.Log(currencyBase[cur])
		x := base + r.normal(0, .05)
		vol := fxVol[cur]
		if vol == 0 {
			vol = .004
		}
		for t := day(2018, 1, 1); !t.After(end); t = t.AddDate(0, 0, 1) {
			if isWeekend(t) || (t.Month() == time.January && t.Day() == 1) || (t.Month() == time.December && t.Day() == 25) {
				continue
			}
			x += .003*(base-x) + r.normal(0, vol)
			fx.Row(cur, Date(t), Dec{math.Exp(x), 8})
		}
	}
	fx.Flush()

	// ---------------------------------------------------------------- accounts and transactions
	acc := newInserter(g.ctx, g.conn, "accounts", "account_id", "customer_id", "account_type", "currency", "opened_on", "closed_on", "credit_limit")
	type account struct {
		id             int
		typ, cur       string
		opened, closed time.Time
		rate           float64 // USD per unit of currency
	}
	perm := r.Perm(len(g.customers))
	holders := perm[:min(c.BankCustomers, len(perm))]
	slices.Sort(holders)
	var txns []txnRow
	var brokerage []account
	accID := 0
	catW := make([]float64, len(merchantCats))
	for i, mc := range merchantCats {
		catW[i] = mc.w
	}
	money := func(usd, rate float64, cur string) float64 {
		v := usd / rate
		if cur == "JPY" {
			return math.Round(v)
		}
		return round2(v)
	}
	for _, ci := range holders {
		cu := g.customers[ci]
		cur := countryCurrency[cu.country]
		rate := currencyBase[cur]
		opened := truncDay(addDays(maxTime(cu.signup, day(2022, 1, 1)), r.between(0, 300)))
		if opened.After(day(2025, 9, 30)) {
			opened = day(2025, 9, 30).AddDate(0, 0, -r.IntN(200))
		}
		var mine []account
		add := func(typ string, offsetDays int) account {
			accID++
			a := account{id: accID, typ: typ, cur: cur, rate: rate, opened: opened.AddDate(0, 0, offsetDays)}
			if r.chance(.05) {
				a.closed = truncDay(r.randTime(a.opened.AddDate(0, 3, 0), end))
				if !a.closed.Before(end) {
					a.closed = time.Time{}
				}
			}
			var closed, limit any
			if !a.closed.IsZero() {
				closed = Date(a.closed)
			}
			if typ == "credit" {
				limit = D2(math.Round(r.between(1000, 25000)/rate/500) * 500)
			}
			acc.Row(a.id, cu.id, typ, cur, Date(a.opened), closed, limit)
			mine = append(mine, a)
			return a
		}
		var checking, savings, credit *account
		if r.chance(.9) {
			a := add("checking", 0)
			checking = &a
		}
		if r.chance(.5) || checking == nil {
			a := add("savings", r.IntN(90))
			savings = &a
		}
		if r.chance(.35) {
			a := add("credit", r.IntN(200))
			credit = &a
		}
		if r.chance(.18) {
			brokerage = append(brokerage, add("brokerage", r.IntN(300)))
		}
		for _, a := range mine {
			if a.typ == "brokerage" {
				txns = append(txns, txnRow{a.id, a.opened.Add(10 * time.Hour), money(r.between(1000, 50000), rate, cur), "deposit", "", ""})
			}
		}

		// month by month, keeping simple balances for savings interest and card repayment
		monthlyPay := r.lognormal(math.Log(4200), .45)
		semiMonthly := r.chance(.5)
		subs := []subscription{}
		for _, s := range subscriptions {
			if r.chance(.35) {
				subs = append(subs, s)
			}
		}
		savingsBal := 0.0
		if savings != nil {
			dep := money(r.between(500, 20000), rate, cur)
			txns = append(txns, txnRow{savings.id, savings.opened.Add(11 * time.Hour), dep, "deposit", "", ""})
			savingsBal = dep
		}
		cardOwed := 0.0
		activity := r.lognormal(0, .45)
		open := func(a *account, t time.Time) bool {
			return a != nil && !t.Before(a.opened) && (a.closed.IsZero() || t.Before(a.closed)) && !t.After(g.end)
		}
		for m := day(opened.Year(), opened.Month(), 1); !m.After(end); m = m.AddDate(0, 1, 0) {
			mEnd := m.AddDate(0, 1, 0).Add(-time.Second)
			if checking != nil {
				payDays := []int{1}
				amt := monthlyPay
				if semiMonthly {
					payDays = []int{1, 15}
					amt /= 2
				}
				for _, pd := range payDays {
					t := m.AddDate(0, 0, pd-1)
					for isWeekend(t) {
						t = t.AddDate(0, 0, 1)
					}
					t = t.Add(time.Duration(6+r.IntN(3)) * time.Hour)
					if open(checking, t) {
						txns = append(txns, txnRow{checking.id, t, money(amt*r.between(.99, 1.01), rate, cur), "salary", "", ""})
					}
				}
				for _, s := range subs {
					t := m.AddDate(0, 0, 2+int(s.usd)%20).Add(3 * time.Hour)
					if open(checking, t) {
						txns = append(txns, txnRow{checking.id, t, -money(s.usd, rate, cur), "purchase", s.merchant, s.category})
					}
				}
				n := int(activity*5 + r.Float64()*2)
				for k := 0; k < n; k++ {
					t := r.randTime(m, mEnd)
					t = truncDay(t).Add(time.Duration(r.weighted(hourWeights))*time.Hour + time.Duration(r.IntN(3600))*time.Second)
					if !open(checking, t) {
						continue
					}
					mc := merchantCats[r.weighted(catW)]
					usd := math.Exp(r.between(math.Log(mc.min), math.Log(mc.max)))
					txns = append(txns, txnRow{checking.id, t, -money(usd, rate, cur), "purchase", r.pick(mc.merchants), mc.name})
					if r.chance(.025) {
						rt := t.AddDate(0, 0, r.intn(2, 20))
						if open(checking, rt) {
							txns = append(txns, txnRow{checking.id, rt, money(usd, rate, cur), "refund", txns[len(txns)-1].merchant, mc.name})
						}
					}
				}
				if r.chance(.8) {
					t := r.randTime(m, mEnd)
					if open(checking, t) {
						txns = append(txns, txnRow{checking.id, t, -money(float64(r.intn(1, 20))*20, rate, cur), "withdrawal", "", ""})
					}
				}
				if r.chance(.06) {
					t := r.randTime(m, mEnd)
					if open(checking, t) {
						txns = append(txns, txnRow{checking.id, t, -money([]float64{12, 12, 35}[r.IntN(3)], rate, cur), "fee", "", ""})
					}
				}
				if r.chance(.05) {
					t := r.randTime(m, mEnd)
					if open(checking, t) {
						txns = append(txns, txnRow{checking.id, t, money(r.between(50, 2000), rate, cur), "deposit", "", ""})
					}
				}
				// move money to savings
				t := m.AddDate(0, 0, 2).Add(9 * time.Hour)
				if savings != nil && r.chance(.6) && open(checking, t) && open(savings, t) {
					amt := money(math.Round(r.between(100, 1500)/50)*50, rate, cur)
					txns = append(txns, txnRow{checking.id, t, -amt, "transfer_out", "", ""},
						txnRow{savings.id, t.Add(time.Second), amt, "transfer_in", "", ""})
					savingsBal += amt
				}
				// pay last month's credit card bill
				t = m.AddDate(0, 0, 19).Add(10 * time.Hour)
				if credit != nil && cardOwed > 0 && open(checking, t) && open(credit, t) {
					amt := round2(cardOwed)
					txns = append(txns, txnRow{checking.id, t, -amt, "transfer_out", "", ""},
						txnRow{credit.id, t.Add(time.Second), amt, "transfer_in", "", ""})
					cardOwed = 0
				}
			}
			if credit != nil {
				n := int(activity*4 + r.Float64()*2)
				for k := 0; k < n; k++ {
					t := r.randTime(m, mEnd)
					if !open(credit, t) {
						continue
					}
					mc := merchantCats[r.weighted(catW)]
					usd := math.Exp(r.between(math.Log(mc.min), math.Log(mc.max)))
					amt := money(usd, rate, cur)
					txns = append(txns, txnRow{credit.id, t, -amt, "purchase", r.pick(mc.merchants), mc.name})
					cardOwed += amt
				}
				if cardOwed > 0 && r.chance(.08) {
					t := mEnd.Add(-time.Hour)
					if open(credit, t) {
						fee := round2(cardOwed * .02)
						txns = append(txns, txnRow{credit.id, t, -fee, "interest", "", ""})
						cardOwed += fee
					}
				}
			}
			if savings != nil {
				t := mEnd.Add(-2 * time.Hour)
				if open(savings, t) && savingsBal > 0 {
					apr := .005
					if m.Year() >= 2023 {
						apr = .035
					}
					interest := round2(savingsBal * apr / 12)
					if cur == "JPY" {
						interest = math.Round(interest)
					}
					if interest > 0 {
						txns = append(txns, txnRow{savings.id, t, interest, "interest", "", ""})
						savingsBal += interest
					}
				}
				if r.chance(.04) {
					t := r.randTime(m, mEnd)
					if open(savings, t) && savingsBal > 100 {
						amt := round2(savingsBal * r.between(.1, .4))
						txns = append(txns, txnRow{savings.id, t, -amt, "withdrawal", "", ""})
						savingsBal -= amt
					}
				}
			}
		}
	}
	acc.Flush()
	slices.SortStableFunc(txns, func(a, b txnRow) int { return a.at.Compare(b.at) })
	tx := newInserter(g.ctx, g.conn, "transactions", "txn_id", "account_id", "txn_at", "amount", "txn_type", "merchant", "merchant_category")
	for i, t := range txns {
		var merch, cat any
		if t.merchant != "" {
			merch, cat = t.merchant, t.category
		}
		tx.Row(i+1, t.account, DateTime(t.at), D2(t.amount), t.typ, merch, cat)
	}
	tx.Flush()

	// ---------------------------------------------------------------- trades
	type tradeRow struct {
		account, stock, qty int
		side                string
		price               float64
		at                  time.Time
	}
	var trades []tradeRow
	perAccount := float64(c.Trades) / math.Max(1, float64(len(brokerage)))
	for _, a := range brokerage {
		n := int(r.lognormal(math.Log(perAccount), .7))
		favs := make([]int, r.intn(3, 12))
		for k := range favs {
			favs[k] = r.IntN(c.Stocks)
		}
		first, ok := dayIdx[a.opened]
		if !ok {
			for first = range days {
				if !days[first].Before(a.opened) {
					break
				}
			}
		}
		lastDay := len(days) - 1
		if !a.closed.IsZero() {
			for lastDay > first && days[lastDay].After(a.closed) {
				lastDay--
			}
		}
		if lastDay <= first {
			continue
		}
		picks := make([]int, n)
		for k := range picks {
			picks[k] = first + r.IntN(lastDay-first+1)
		}
		slices.Sort(picks)
		pos := map[int]int{}
		for _, di := range picks {
			s := favs[r.IntN(len(favs))]
			px := closes[s][di]
			if px == 0 {
				continue
			}
			side := "buy"
			qty := max(1, int(r.lognormal(math.Log(4000/px+1), .8)))
			if pos[s] > 0 && r.chance(.4) {
				side = "sell"
				qty = min(pos[s], max(1, int(float64(pos[s])*r.between(.2, 1.01))))
				pos[s] -= qty
			} else {
				pos[s] += qty
			}
			at := days[di].Add(9*time.Hour + 30*time.Minute + time.Duration(r.Int64N(int64(6*time.Hour+29*time.Minute))))
			trades = append(trades, tradeRow{a.id, s + 1, qty, side, px * (1 + r.normal(0, .003)), at})
		}
	}
	slices.SortStableFunc(trades, func(a, b tradeRow) int { return a.at.Compare(b.at) })
	tr := newInserter(g.ctx, g.conn, "trades", "trade_id", "account_id", "stock_id", "side", "quantity", "price", "executed_at")
	for i, t := range trades {
		tr.Row(i+1, t.account, t.stock, t.side, t.qty, Dec{t.price, 4}, DateTime(t.at))
	}
	tr.Flush()

	// ---------------------------------------------------------------- loans
	ln := newInserter(g.ctx, g.conn, "loans", "loan_id", "customer_id", "loan_type", "principal", "annual_rate", "term_months", "start_date", "status")
	type loanRow struct {
		cust      int
		typ       string
		principal float64
		rate      float64
		term      int
		start     time.Time
	}
	var loans []loanRow
	for k := 0; k < c.Loans; k++ {
		cu := g.customers[r.IntN(len(g.customers))]
		l := loanRow{cust: cu.id, start: truncDay(r.randTime(day(2015, 1, 1), end.AddDate(0, -1, 0)))}
		hike := 0.0
		if l.start.Year() >= 2023 {
			hike = .025
		}
		switch x := r.Float64(); {
		case x < .4:
			l.typ, l.principal, l.term, l.rate = "personal", math.Round(r.between(2000, 40000)/100)*100, []int{24, 36, 48, 60}[r.IntN(4)], r.between(.07, .18)
		case x < .7:
			l.typ, l.principal, l.term, l.rate = "auto", math.Round(r.between(8000, 60000)/100)*100, []int{36, 48, 60, 72}[r.IntN(4)], r.between(.04, .10)
		case x < .85:
			l.typ, l.principal, l.term, l.rate = "mortgage", math.Round(r.between(120000, 900000)/1000)*1000, []int{180, 240, 360, 360}[r.IntN(4)], r.between(.03, .075)
		default:
			l.typ, l.principal, l.term, l.rate = "student", math.Round(r.between(5000, 80000)/100)*100, []int{60, 120, 120}[r.IntN(3)], r.between(.035, .08)
		}
		l.rate += hike
		loans = append(loans, l)
	}
	slices.SortStableFunc(loans, func(a, b loanRow) int { return a.start.Compare(b.start) })
	for i, l := range loans {
		status := "active"
		if l.start.AddDate(0, l.term, 0).Before(end) {
			status = "paid_off"
			if r.chance(.07) {
				status = "defaulted"
			}
		} else if r.chance(.04) {
			status = "defaulted"
		} else if r.chance(.06) && l.typ != "mortgage" {
			status = "paid_off" // paid early
		}
		ln.Row(i+1, l.cust, l.typ, D2(l.principal), Dec{l.rate, 4}, l.term, Date(l.start), status)
	}
	ln.Flush()
	g.logf("finance: %d stocks, %d prices, %d fx rates, %d accounts, %d transactions, %d trades, %d loans",
		st.Total, sp.Total, fx.Total, acc.Total, tx.Total, tr.Total, ln.Total)
}

func (g *gen) calendar() {
	cal := newInserter(g.ctx, g.conn, "calendar", "date_key", "year", "quarter", "month", "month_name", "day_of_month",
		"day_of_week", "day_name", "iso_week", "is_weekend", "is_holiday")
	hol := map[int]map[time.Time]bool{}
	for t := day(2012, 1, 1); t.Year() <= 2030; t = t.AddDate(0, 0, 1) {
		if hol[t.Year()] == nil {
			hol[t.Year()] = usMarketHolidays(t.Year())
		}
		_, wk := t.ISOWeek()
		dow := int(t.Weekday())
		if dow == 0 {
			dow = 7
		}
		cal.Row(Date(t), t.Year(), (int(t.Month())+2)/3, int(t.Month()), t.Month().String(), t.Day(), dow, t.Weekday().String(), wk, isWeekend(t), hol[t.Year()][t])
	}
	cal.Flush()
}
