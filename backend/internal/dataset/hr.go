package dataset

import (
	"fmt"
	"math"
	"slices"
	"time"
)

type office struct {
	city, country, region string
	opened                time.Time
	payFactor             float64
}

var offices = []office{
	{"New York", "United States", "AMER", day(2012, 3, 1), 1.15},
	{"San Francisco", "United States", "AMER", day(2013, 6, 1), 1.2},
	{"Austin", "United States", "AMER", day(2016, 2, 1), 1.0},
	{"Toronto", "Canada", "AMER", day(2017, 9, 1), .8},
	{"São Paulo", "Brazil", "AMER", day(2019, 4, 1), .45},
	{"London", "United Kingdom", "EMEA", day(2014, 5, 1), .9},
	{"Berlin", "Germany", "EMEA", day(2016, 10, 1), .8},
	{"Amsterdam", "Netherlands", "EMEA", day(2018, 3, 1), .85},
	{"Bengaluru", "India", "APAC", day(2015, 7, 1), .35},
	{"Singapore", "Singapore", "APAC", day(2017, 1, 15), .85},
	{"Sydney", "Australia", "APAC", day(2018, 8, 1), .85},
	{"Tokyo", "Japan", "APAC", day(2020, 2, 1), .8},
}

type deptDef struct {
	name   string
	share  float64
	payMul float64
	ladder [7]string // levels 1..7
}

var deptDefs = []deptDef{
	{"Engineering", 22, 1.2, [7]string{"Software Engineer I", "Software Engineer II", "Senior Software Engineer", "Staff Engineer", "Engineering Manager", "Director of Engineering", "VP of Engineering"}},
	{"Product", 4, 1.15, [7]string{"Associate Product Manager", "Product Manager", "Senior Product Manager", "Principal Product Manager", "Group Product Manager", "Director of Product", "VP of Product"}},
	{"Design", 4, 1.0, [7]string{"Junior Designer", "Product Designer", "Senior Product Designer", "Lead Designer", "Design Manager", "Director of Design", "VP of Design"}},
	{"Data", 6, 1.15, [7]string{"Data Analyst", "Data Scientist", "Senior Data Scientist", "Staff Data Scientist", "Analytics Manager", "Director of Data", "VP of Data"}},
	{"Sales", 11, 1.0, [7]string{"Sales Development Rep", "Account Executive", "Senior Account Executive", "Enterprise Account Executive", "Sales Manager", "Director of Sales", "VP of Sales"}},
	{"Marketing", 6, .95, [7]string{"Marketing Coordinator", "Marketing Specialist", "Senior Marketing Specialist", "Marketing Lead", "Marketing Manager", "Director of Marketing", "VP of Marketing"}},
	{"Customer Support", 15, .7, [7]string{"Support Associate", "Support Specialist", "Senior Support Specialist", "Support Team Lead", "Support Manager", "Director of Support", "VP of Customer Experience"}},
	{"Finance", 3, 1.0, [7]string{"Accounting Associate", "Financial Analyst", "Senior Financial Analyst", "Finance Lead", "Finance Manager", "Controller", "VP of Finance"}},
	{"People", 2.5, .9, [7]string{"HR Coordinator", "HR Generalist", "Senior HR Business Partner", "HR Lead", "People Manager", "Director of People", "VP of People"}},
	{"Legal", 1.5, 1.15, [7]string{"Paralegal", "Legal Counsel", "Senior Legal Counsel", "Lead Counsel", "Legal Manager", "Director of Legal", "General Counsel"}},
	{"Operations", 7, .85, [7]string{"Operations Associate", "Operations Analyst", "Senior Operations Analyst", "Operations Lead", "Operations Manager", "Director of Operations", "VP of Operations"}},
	{"IT", 3, .95, [7]string{"IT Support Technician", "Systems Administrator", "Senior Systems Engineer", "IT Lead", "IT Manager", "Director of IT", "VP of IT"}},
	{"Logistics", 14, .7, [7]string{"Warehouse Associate", "Logistics Coordinator", "Senior Logistics Coordinator", "Logistics Lead", "Warehouse Manager", "Director of Logistics", "VP of Supply Chain"}},
}

// level salary bands in USD before department multiplier: level 1..8
var levelBand = [9][2]float64{{}, {45000, 65000}, {60000, 85000}, {85000, 120000}, {110000, 150000}, {120000, 165000}, {150000, 210000}, {200000, 300000}, {350000, 500000}}

type employee struct {
	id, dept, level, mgr, office int
	jobByLevel                   map[int]int
	hire, term                   time.Time
	startLevel                   int
	perf                         float64
	contractor, partTime         bool
}

func (g *gen) hr() {
	r := newRNG(g.opt.Seed + 2)
	end := truncDay(g.end)

	off := newInserter(g.ctx, g.conn, "offices", "office_id", "city", "country", "region", "opened_on")
	for i, o := range offices {
		off.Row(i+1, o.city, o.country, o.region, Date(o.opened))
	}
	off.Flush()

	// jobs: ladder per department plus executive roles
	jobs := newInserter(g.ctx, g.conn, "jobs", "job_id", "title", "job_family", "level", "min_salary", "max_salary")
	jobID := 0
	jobIDs := make([][8]int, len(deptDefs)) // [dept][level] -> job id
	for di, d := range deptDefs {
		for lvl := 1; lvl <= 7; lvl++ {
			jobID++
			jobIDs[di][lvl] = jobID
			jobs.Row(jobID, d.ladder[lvl-1], d.name, lvl, D2(math.Round(levelBand[lvl][0]*d.payMul/1000)*1000), D2(math.Round(levelBand[lvl][1]*d.payMul/1000)*1000))
		}
	}
	ceoJob := jobID + 1
	jobs.Row(ceoJob, "Chief Executive Officer", "Executive", 8, D2(levelBand[8][0]), D2(levelBand[8][1]))
	eaJob := jobID + 2
	jobs.Row(eaJob, "Executive Assistant", "Executive", 2, D2(60000), D2(90000))
	cosJob := jobID + 3
	jobs.Row(cosJob, "Chief of Staff", "Executive", 5, D2(150000), D2(220000))
	jobs.Flush()

	execDept := len(deptDefs) // department id index for "Executive"
	dep := newInserter(g.ctx, g.conn, "departments", "department_id", "name", "budget")
	for i, d := range deptDefs {
		dep.Row(i+1, d.name, D2(math.Round(d.share*r.between(.9, 1.1)*3.2e6/1000)*1000))
	}
	dep.Row(execDept+1, "Executive", D2(4.5e6))
	dep.Flush()

	// ---------------------------------------------------------------- build the org
	var emps []*employee
	newEmp := func(dept, level, mgr int, hireFrom, hireTo time.Time) *employee {
		e := &employee{id: len(emps) + 1, dept: dept, level: level, mgr: mgr, perf: r.normal(0, .8)}
		e.office = r.weighted([]float64{14, 10, 8, 5, 4, 9, 6, 4, 12, 4, 3, 3})
		e.hire = truncDay(r.randTime(hireFrom, hireTo))
		if o := offices[e.office].opened; e.hire.Before(o) {
			e.hire = o.AddDate(0, 0, r.IntN(120))
		}
		for e.hire.Weekday() != time.Monday && e.hire.Weekday() != time.Wednesday {
			e.hire = e.hire.AddDate(0, 0, 1)
		}
		emps = append(emps, e)
		return e
	}
	ceo := newEmp(execDept, 8, 0, day(2012, 1, 2), day(2012, 1, 3))
	ceo.office = 0
	ceo.hire = day(2012, 1, 2)
	newEmp(execDept, 2, ceo.id, day(2012, 2, 1), day(2016, 1, 1))
	newEmp(execDept, 2, ceo.id, day(2017, 2, 1), day(2022, 1, 1))
	newEmp(execDept, 5, ceo.id, day(2014, 1, 1), day(2019, 1, 1))

	total := g.counts.Employees
	shareSum := 0.0
	for _, d := range deptDefs {
		shareSum += d.share
	}
	for di, d := range deptDefs {
		n := int(math.Round(d.share / shareSum * float64(total-4)))
		vp := newEmp(di, 7, ceo.id, day(2012, 3, 1), day(2017, 1, 1))
		nDir := max(1, n/160)
		nMgr := max(1, n/9)
		var dirs, mgrs []*employee
		for k := 0; k < nDir; k++ {
			dirs = append(dirs, newEmp(di, 6, vp.id, day(2013, 1, 1), day(2021, 1, 1)))
		}
		for k := 0; k < nMgr; k++ {
			lvl := 5
			mgrs = append(mgrs, newEmp(di, lvl, dirs[k%len(dirs)].id, day(2014, 1, 1), day(2024, 1, 1)))
		}
		for k := 0; k < n-1-nDir-nMgr; k++ {
			mgr := mgrs[r.IntN(len(mgrs))].id
			if r.chance(.03) {
				mgr = dirs[r.IntN(len(dirs))].id
			}
			// growth: more recent hires
			u := math.Pow(r.Float64(), .6)
			from := day(2014, 1, 1).Add(time.Duration(u * float64(end.Sub(day(2014, 1, 1)))))
			e := newEmp(di, 1, mgr, from, from.AddDate(0, 0, 1))
			e.level = []int{1, 1, 2, 2, 2, 3, 3, 4}[r.IntN(8)]
			switch x := r.Float64(); {
			case x < .07:
				e.contractor = true
			case x < .12:
				e.partTime = true
			}
			if r.chance(.33) {
				years := end.Sub(e.hire).Hours() / 24 / 365
				e.term = truncDay(addDays(e.hire, math.Min(r.ExpFloat64()*900+60, years*365-1)))
				if !e.term.Before(end) || !e.term.After(e.hire) {
					e.term = time.Time{}
				}
			}
		}
	}

	// promotions decide the starting level; a few managers and directors leave too
	for _, e := range emps {
		e.startLevel = e.level
		if e.level <= 4 {
			years := end.Sub(e.hire).Hours() / 24 / 365
			for p := 0; p < int(years/2.2) && e.startLevel > 1; p++ {
				if r.chance(.45) {
					e.startLevel--
				}
			}
		}
		if (e.level == 5 || e.level == 6) && r.chance(.06) {
			e.term = truncDay(r.randTime(e.hire.AddDate(1, 0, 0), end))
			if !e.term.Before(end) {
				e.term = time.Time{}
			}
		}
	}

	emp := newInserter(g.ctx, g.conn, "employees", "employee_id", "first_name", "last_name", "email", "gender", "birth_date",
		"hire_date", "termination_date", "department_id", "job_id", "manager_id", "office_id", "employment_type")
	usedEmail := map[string]int{}
	for _, e := range emps {
		gender, first := "F", r.pick(femaleNames)
		if r.chance(.53) {
			gender, first = "M", r.pick(maleNames)
		}
		last := r.pick(lastNames)
		local := emailLocal(first, last)
		usedEmail[local]++
		if usedEmail[local] > 1 {
			local += fmt.Sprint(usedEmail[local])
		}
		ageAtHire := 22 + float64(e.level)*2.6 + r.ExpFloat64()*5
		birth := truncDay(e.hire.AddDate(-int(ageAtHire), -r.IntN(12), -r.IntN(28)))
		var term any
		if !e.term.IsZero() {
			term = Date(e.term)
		}
		job := ceoJob
		switch {
		case e.dept == execDept && e.level == 2:
			job = eaJob
		case e.dept == execDept && e.level == 5:
			job = cosJob
		case e.dept < execDept:
			job = jobIDs[e.dept][e.level]
		}
		var mgr any
		if e.mgr != 0 {
			mgr = e.mgr
		}
		et := "full_time"
		if e.contractor {
			et = "contractor"
		} else if e.partTime {
			et = "part_time"
		}
		emp.Row(e.id, first, last, local+"@northbeam.io", gender, Date(birth), Date(e.hire), term, e.dept+1, job, mgr, e.office+1, et)
	}
	emp.Flush()

	// ---------------------------------------------------------------- salaries
	sal := newInserter(g.ctx, g.conn, "salaries", "employee_id", "salary", "from_date", "to_date")
	for _, e := range emps {
		mul := 1.0
		if e.dept < execDept {
			mul = deptDefs[e.dept].payMul
		}
		mul *= offices[e.office].payFactor
		if e.partTime {
			mul *= .55
		}
		lvl := e.startLevel
		band := levelBand[lvl]
		salary := (band[0] + r.between(0, .55)*(band[1]-band[0])) * mul
		salary = math.Round(salary/100) * 100
		from := e.hire
		stop := end
		if !e.term.IsZero() {
			stop = e.term
		}
		for {
			// raises land on April 1st, once someone has been on their salary for 6+ months
			next := day(from.Year(), time.April, 1)
			for next.Before(from.AddDate(0, 6, 0)) {
				next = next.AddDate(1, 0, 0)
			}
			if !next.Before(stop) {
				var to any
				if !e.term.IsZero() {
					to = Date(e.term)
				}
				sal.Row(e.id, D2(salary), Date(from), to)
				break
			}
			sal.Row(e.id, D2(salary), Date(from), Date(next))
			raise := clamp(r.normal(.035+.01*e.perf, .015), 0, .09)
			if lvl < e.level && r.chance(.5) {
				lvl++
				raise += .1 + r.between(0, .05)
			}
			salary = math.Round(salary*(1+raise)/100) * 100
			from = next
		}
	}
	sal.Flush()

	// ---------------------------------------------------------------- performance reviews
	perf := newInserter(g.ctx, g.conn, "performance_reviews", "review_id", "employee_id", "review_year", "rating", "reviewer_id")
	revID := 0
	for y := 2012; y <= end.Year(); y++ {
		cut := day(y, time.December, 31)
		for _, e := range emps {
			if e.hire.After(day(y, time.October, 1)) || (!e.term.IsZero() && e.term.Before(cut)) || e.contractor {
				continue
			}
			score := 3.3 + e.perf*.7 + r.normal(0, .7)
			rating := int(math.Round(clamp(score, 1, 5)))
			var reviewer any
			if e.mgr != 0 {
				reviewer = e.mgr
			}
			revID++
			perf.Row(revID, e.id, y, rating, reviewer)
		}
	}
	perf.Flush()

	// ---------------------------------------------------------------- leave requests
	type lt struct {
		name     string
		w        float64
		min, max int
	}
	types := []lt{{"vacation", 60, 1, 14}, {"sick", 25, 1, 5}, {"parental", 3, 30, 120}, {"unpaid", 7, 1, 20}, {"bereavement", 5, 1, 5}}
	tw := make([]float64, len(types))
	for i, t := range types {
		tw[i] = t.w
	}
	lv := newInserter(g.ctx, g.conn, "leave_requests", "leave_id", "employee_id", "leave_type", "start_date", "end_date", "status")
	type leaveRow struct {
		emp        int
		typ        string
		start, end time.Time
		status     string
	}
	var leaves []leaveRow
	for _, e := range emps {
		stop := end
		if !e.term.IsZero() {
			stop = e.term
		}
		years := stop.Sub(e.hire).Hours() / 24 / 365
		n := int(years*1.3 + r.Float64())
		for k := 0; k < n; k++ {
			t := types[r.weighted(tw)]
			start := truncDay(r.randTime(e.hire.AddDate(0, 1, 0), stop.AddDate(0, 2, 0)))
			for isWeekend(start) {
				start = start.AddDate(0, 0, 1)
			}
			if start.After(end.AddDate(0, 2, 0)) || (!e.term.IsZero() && start.After(e.term)) {
				continue
			}
			days := r.intn(t.min, t.max)
			status := "approved"
			switch {
			case start.After(end.AddDate(0, 0, -20)) && r.chance(.6):
				status = "pending"
			case r.chance(.08):
				status = "rejected"
			}
			leaves = append(leaves, leaveRow{e.id, t.name, start, start.AddDate(0, 0, days-1), status})
		}
	}
	slices.SortStableFunc(leaves, func(a, b leaveRow) int { return a.start.Compare(b.start) })
	for i, l := range leaves {
		lv.Row(i+1, l.emp, l.typ, Date(l.start), Date(l.end), l.status)
	}
	lv.Flush()
	g.logf("hr: %d employees, %d salary rows, %d reviews, %d leave requests", emp.Total, sal.Total, perf.Total, lv.Total)
}
