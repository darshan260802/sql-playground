package dataset

import (
	"math"
	"math/rand/v2"
	"sort"
	"strings"
	"time"
)

var femaleNames = strings.Fields(`Emma Olivia Ava Sophia Isabella Mia Charlotte Amelia Harper Evelyn Abigail Emily
Elizabeth Sofia Avery Ella Scarlett Grace Chloe Victoria Riley Aria Lily Aubrey Zoey Penelope Lillian Addison Layla
Natalie Camila Hannah Brooklyn Zoe Nora Leah Savannah Audrey Claire Eleanor Skylar Ellie Samantha Stella Violet
Mila Allison Anna Hazel Aaliyah Ariana Lucy Caroline Sarah Kennedy Sadie Gabriella Maya Priya Ananya Diya Aisha
Fatima Sara Neha Pooja Kavya Isha Meera Riya Yuki Hana Sakura Mei Lucia Valentina Camille Léa Chloé Manon Lena
Laura Julia Marie Sofía Martina Giulia Chiara Ana Beatriz Mariana Fernanda Ximena Ingrid Freya Nina Elena Olga
Zara Amara Ling Siobhan Aoife Noor Leila Hiroko Jessica Ashley Megan Rachel Nicole`)

var maleNames = strings.Fields(`Liam Noah William James Oliver Benjamin Elijah Lucas Mason Logan Alexander Ethan
Jacob Michael Daniel Henry Jackson Sebastian Aiden Matthew Samuel David Joseph Carter Owen Wyatt John Jack Luke
Jayden Dylan Grayson Levi Isaac Gabriel Julian Mateo Anthony Lincoln Joshua Christopher Andrew Theodore Caleb Ryan
Asher Nathan Thomas Leo Isaiah Charles Hudson Christian Hunter Connor Eli Ezra Aaron Arjun Rohan Aarav Vihaan Aditya
Rahul Karan Vikram Sanjay Amit Kenji Haruto Ren Takumi Wei Jun Lukas Felix Maximilian Jonas Paul Louis Hugo Arthur
Jules Pablo Diego Javier Alejandro Matteo Lorenzo Francesco Leonardo João Pedro Miguel Rafael Santiago Emiliano
Omar Yusuf Ali Hamza Kwame Ibrahim Sean Patrick Declan Erik Mikael Tariq Kevin Brian Justin`)

var lastNames = strings.Fields(`Smith Johnson Williams Brown Jones Garcia Miller Davis Rodriguez Martinez Hernandez
Lopez Gonzalez Wilson Anderson Thomas Taylor Moore Jackson Martin Lee Perez Thompson White Harris Sanchez Clark
Ramirez Lewis Robinson Walker Young Allen King Wright Scott Torres Nguyen Hill Flores Green Adams Nelson Baker Hall
Rivera Campbell Mitchell Carter Roberts Patel Shah Sharma Singh Kumar Gupta Mehta Reddy Iyer Nair Desai Joshi Kapoor
Chopra Rao Das Bose Tanaka Suzuki Sato Takahashi Watanabe Yamamoto Kim Park Choi Wang Li Zhang Liu Chen Yang Huang
Müller Schmidt Schneider Fischer Weber Meyer Wagner Becker Schulz Hoffmann Bernard Dubois Robert Richard Petit
Durand Leroy Moreau Rossi Russo Ferrari Esposito Bianchi Romano Colombo Silva Santos Oliveira Souza Costa Pereira
Almeida Fernández García López Martínez González Rodríguez Jansen Bakker Visser O'Brien Murphy Kelly Walsh Byrne
MacDonald Stewart Thomson Andersson Johansson Nielsen Hansen Kowalski Nowak Ivanov Smirnov Okafor Mensah Haddad
Khan Ahmed Hassan O'Neill D'Souza Fitzgerald`)

type city struct {
	Name, State, Country string
	W                    float64
}

var cities = []city{
	{"New York", "New York", "United States", 4}, {"Los Angeles", "California", "United States", 3},
	{"Chicago", "Illinois", "United States", 2.5}, {"Houston", "Texas", "United States", 2},
	{"Phoenix", "Arizona", "United States", 1.5}, {"Philadelphia", "Pennsylvania", "United States", 1.5},
	{"San Antonio", "Texas", "United States", 1}, {"San Diego", "California", "United States", 1.2},
	{"Dallas", "Texas", "United States", 1.5}, {"Austin", "Texas", "United States", 1.5},
	{"San Francisco", "California", "United States", 2}, {"Seattle", "Washington", "United States", 2},
	{"Denver", "Colorado", "United States", 1.5}, {"Boston", "Massachusetts", "United States", 1.8},
	{"Atlanta", "Georgia", "United States", 1.5}, {"Miami", "Florida", "United States", 1.5},
	{"Portland", "Oregon", "United States", 1}, {"Minneapolis", "Minnesota", "United States", 1},
	{"Nashville", "Tennessee", "United States", 1}, {"Detroit", "Michigan", "United States", 0.8},
	{"Mumbai", "Maharashtra", "India", 3}, {"Bengaluru", "Karnataka", "India", 3}, {"Delhi", "Delhi", "India", 2.5},
	{"Hyderabad", "Telangana", "India", 2}, {"Pune", "Maharashtra", "India", 1.5}, {"Chennai", "Tamil Nadu", "India", 1.5},
	{"Ahmedabad", "Gujarat", "India", 1}, {"Kolkata", "West Bengal", "India", 0.8},
	{"London", "England", "United Kingdom", 5}, {"Manchester", "England", "United Kingdom", 1.5},
	{"Birmingham", "England", "United Kingdom", 1.2}, {"Leeds", "England", "United Kingdom", 0.8},
	{"Glasgow", "Scotland", "United Kingdom", 0.7}, {"Edinburgh", "Scotland", "United Kingdom", 0.6},
	{"Bristol", "England", "United Kingdom", 0.5},
	{"Berlin", "Berlin", "Germany", 2.5}, {"Munich", "Bavaria", "Germany", 1.8}, {"Hamburg", "Hamburg", "Germany", 1.2},
	{"Frankfurt", "Hesse", "Germany", 0.9}, {"Cologne", "North Rhine-Westphalia", "Germany", 0.8},
	{"Toronto", "Ontario", "Canada", 2.5}, {"Vancouver", "British Columbia", "Canada", 1.5},
	{"Montreal", "Quebec", "Canada", 1.3}, {"Calgary", "Alberta", "Canada", 0.8},
	{"Paris", "Île-de-France", "France", 2.8}, {"Lyon", "Auvergne-Rhône-Alpes", "France", 0.9},
	{"Marseille", "Provence-Alpes-Côte d'Azur", "France", 0.7}, {"Toulouse", "Occitanie", "France", 0.6},
	{"Sydney", "New South Wales", "Australia", 2}, {"Melbourne", "Victoria", "Australia", 1.8},
	{"Brisbane", "Queensland", "Australia", 0.7}, {"Perth", "Western Australia", "Australia", 0.5},
	{"Tokyo", "Tokyo", "Japan", 2}, {"Osaka", "Osaka", "Japan", 1},
	{"São Paulo", "São Paulo", "Brazil", 1.8}, {"Rio de Janeiro", "Rio de Janeiro", "Brazil", 1.2},
	{"Mexico City", "Mexico City", "Mexico", 1.3}, {"Guadalajara", "Jalisco", "Mexico", 0.7},
	{"Madrid", "Madrid", "Spain", 1.4}, {"Barcelona", "Catalonia", "Spain", 1.1},
	{"Rome", "Lazio", "Italy", 1}, {"Milan", "Lombardy", "Italy", 1},
	{"Amsterdam", "North Holland", "Netherlands", 1}, {"Rotterdam", "South Holland", "Netherlands", 0.5},
	{"Singapore", "", "Singapore", 1},
}

var countryCurrency = map[string]string{
	"United States": "USD", "India": "INR", "United Kingdom": "GBP", "Germany": "EUR", "Canada": "CAD",
	"France": "EUR", "Australia": "AUD", "Japan": "JPY", "Brazil": "BRL", "Mexico": "MXN", "Spain": "EUR",
	"Italy": "EUR", "Netherlands": "EUR", "Singapore": "SGD",
}

var phonePrefix = map[string]string{
	"United States": "+1", "India": "+91", "United Kingdom": "+44", "Germany": "+49", "Canada": "+1",
	"France": "+33", "Australia": "+61", "Japan": "+81", "Brazil": "+55", "Mexico": "+52", "Spain": "+34",
	"Italy": "+39", "Netherlands": "+31", "Singapore": "+65",
}

// Base USD value of one unit of each currency, used for FX rates and to size amounts.
var currencyBase = map[string]float64{
	"USD": 1, "EUR": 1.09, "GBP": 1.27, "INR": 0.0121, "CAD": 0.74, "AUD": 0.67, "JPY": 0.0071,
	"BRL": 0.19, "MXN": 0.054, "SGD": 0.74, "CHF": 1.11, "CNY": 0.139,
}

var emailDomains = []string{"gmail.com", "gmail.com", "gmail.com", "yahoo.com", "outlook.com", "hotmail.com", "icloud.com", "proton.me", "mail.com"}

var asciiFold = strings.NewReplacer(
	"é", "e", "è", "e", "ê", "e", "ë", "e", "á", "a", "à", "a", "â", "a", "ã", "a", "ä", "ae",
	"ö", "oe", "ü", "ue", "ó", "o", "ò", "o", "ô", "o", "í", "i", "ì", "i", "ú", "u", "ñ", "n",
	"ç", "c", "ß", "ss", "'", "", " ", "", "É", "e", "Á", "a", "Ó", "o", "Í", "i", "Ú", "u", "Ü", "ue", "Ö", "oe",
)

func emailLocal(first, last string) string {
	return strings.ToLower(asciiFold.Replace(first) + "." + asciiFold.Replace(last))
}

// --------------------------------------------------------------------------- random helpers

type rng struct{ *rand.Rand }

func newRNG(seed uint64) rng { return rng{rand.New(rand.NewPCG(seed, seed*0x9E3779B97F4A7C15+1))} }

func (r rng) between(lo, hi float64) float64 { return lo + r.Float64()*(hi-lo) }
func (r rng) intn(lo, hi int) int            { return lo + r.IntN(hi-lo+1) } // inclusive
func (r rng) chance(p float64) bool          { return r.Float64() < p }
func (r rng) pick(xs []string) string        { return xs[r.IntN(len(xs))] }
func (r rng) normal(mean, sd float64) float64 {
	return mean + sd*r.NormFloat64()
}
func (r rng) lognormal(mu, sigma float64) float64 { return math.Exp(mu + sigma*r.NormFloat64()) }

// weighted picks an index according to weights.
func (r rng) weighted(w []float64) int {
	total := 0.0
	for _, x := range w {
		total += x
	}
	t := r.Float64() * total
	for i, x := range w {
		t -= x
		if t < 0 {
			return i
		}
	}
	return len(w) - 1
}

// sampler draws indexes from a fixed weight distribution in O(log n).
type sampler struct{ cum []float64 }

func newSampler(w []float64) sampler {
	cum := make([]float64, len(w))
	s := 0.0
	for i, x := range w {
		s += x
		cum[i] = s
	}
	return sampler{cum}
}

func (s sampler) draw(r rng) int {
	t := r.Float64() * s.cum[len(s.cum)-1]
	return sort.SearchFloat64s(s.cum, t)
}

// --------------------------------------------------------------------------- dates

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func truncDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func addDays(t time.Time, d float64) time.Time {
	return t.Add(time.Duration(d * 24 * float64(time.Hour)))
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// randTime returns a uniformly random instant in [a, b).
func (r rng) randTime(a, b time.Time) time.Time {
	if !b.After(a) {
		return a
	}
	return a.Add(time.Duration(r.Int64N(int64(b.Sub(a)))))
}

// nthWeekday returns the n-th weekday of a month (n = -1 for the last one).
func nthWeekday(y int, m time.Month, wd time.Weekday, n int) time.Time {
	if n < 0 {
		t := day(y, m+1, 1).AddDate(0, 0, -1)
		for t.Weekday() != wd {
			t = t.AddDate(0, 0, -1)
		}
		return t
	}
	t := day(y, m, 1)
	for t.Weekday() != wd {
		t = t.AddDate(0, 0, 1)
	}
	return t.AddDate(0, 0, 7*(n-1))
}

func easter(y int) time.Time {
	a := y % 19
	b := y / 100
	c := y % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	dd := ((h + l - 7*m + 114) % 31) + 1
	return day(y, time.Month(month), dd)
}

func observed(t time.Time) time.Time {
	switch t.Weekday() {
	case time.Saturday:
		return t.AddDate(0, 0, -1)
	case time.Sunday:
		return t.AddDate(0, 0, 1)
	}
	return t
}

// usMarketHolidays returns NYSE-style holidays for a year.
func usMarketHolidays(y int) map[time.Time]bool {
	h := map[time.Time]bool{
		nthWeekday(y, time.January, time.Monday, 3):    true,
		nthWeekday(y, time.February, time.Monday, 3):   true,
		easter(y).AddDate(0, 0, -2):                    true,
		nthWeekday(y, time.May, time.Monday, -1):       true,
		observed(day(y, time.July, 4)):                 true,
		nthWeekday(y, time.September, time.Monday, 1):  true,
		nthWeekday(y, time.November, time.Thursday, 4): true,
		observed(day(y, time.December, 25)):            true,
	}
	if ny := day(y, time.January, 1); ny.Weekday() != time.Saturday {
		h[observed(ny)] = true
	}
	if y >= 2022 {
		h[observed(day(y, time.June, 19))] = true
	}
	return h
}

func isWeekend(t time.Time) bool { return t.Weekday() == time.Saturday || t.Weekday() == time.Sunday }

// tradingDays lists market days in [from, to].
func tradingDays(from, to time.Time) []time.Time {
	var out []time.Time
	hol := map[int]map[time.Time]bool{}
	for t := from; !t.After(to); t = t.AddDate(0, 0, 1) {
		if isWeekend(t) {
			continue
		}
		if hol[t.Year()] == nil {
			hol[t.Year()] = usMarketHolidays(t.Year())
		}
		if hol[t.Year()][t] {
			continue
		}
		out = append(out, t)
	}
	return out
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
