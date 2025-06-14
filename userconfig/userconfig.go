package userconfig

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ptgott/one-newsletter/linksrc"
	"github.com/rs/zerolog/log"

	"github.com/ptgott/one-newsletter/email"

	yaml "gopkg.in/yaml.v2"
)

// Meta represents all current config options that the application can use,
// i.e., after validation and parsing
type Meta struct {
	Scraping      Scraping         `yaml:"scraping"`
	EmailSettings email.UserConfig `yaml:"email"`
	LinkSources   []linksrc.Config `yaml:"link_sources"`
}

// Weekdays is a bitmap indicating the days of the week in which to send a
// newsletter.
type Weekdays int

const (
	Monday Weekdays = 1 << iota
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
	Sunday
)

var allDays [7]Weekdays = [7]Weekdays{
	Monday,
	Tuesday,
	Wednesday,
	Thursday,
	Friday,
	Saturday,
	Sunday,
}

var daysToTime map[Weekdays]time.Weekday = map[Weekdays]time.Weekday{
	Monday:    time.Monday,
	Tuesday:   time.Tuesday,
	Wednesday: time.Wednesday,
	Thursday:  time.Thursday,
	Friday:    time.Friday,
	Saturday:  time.Saturday,
	Sunday:    time.Sunday,
}

const DefaultScheduleName = "newsletter"

type NotificationSchedule struct {
	Weekdays Weekdays
	Hour     int
}

// Moment specifies attributes of a time as returned by methods of
// time.Time. It is used to determine whether the current time belongs to a
// notification schedule.
type Moment struct {
	Day  time.Weekday
	Hour int
}

// moments gets the notificationMoments that correspond to a
// NotificationSchedule. Consumers can map notificationMoments to, for example,
// the originating NotificationSchedules or configurations as well as determine
// whether a schedule belongs to the current time.
func moments(s NotificationSchedule) []Moment {
	res := []Moment{}
	for _, d := range allDays {
		if s.Weekdays&d != d {
			continue
		}
		res = append(res, Moment{
			Day:  daysToTime[d],
			Hour: s.Hour,
		})
	}
	return res
}

// ScheduleStore tracks handles for NotificationSchedules, mapping moments to
// them so we can look up a given schedule by the current moment.
type ScheduleStore struct {
	moments    map[Moment][]string
	mu         *sync.Mutex
	lastMoment Moment
	// lastDate is the last date we queried a moment in RFC 3339 format,
	// e.g., 2025-06-05.
	lastDate string
}

// NewScheduleStore initializes an empty ScheduleStore.
func NewScheduleStore() *ScheduleStore {
	return &ScheduleStore{
		moments: make(map[Moment][]string),
		mu:      &sync.Mutex{},
	}
}

// Add includes sched in the ScheduleStore so we can look it up later by moment.
func (s *ScheduleStore) Add(handle string, sched NotificationSchedule) {
	for _, m := range moments(sched) {
		if _, ok := s.moments[m]; !ok {
			s.moments[m] = []string{}
		}
		s.moments[m] = append(s.moments[m], handle)
	}
}

// Get returns a slice of NotificationSchedule handles that match the given
// notification moment.
func (s *ScheduleStore) Get(t time.Time) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := Moment{
		Day:  t.Weekday(),
		Hour: t.Hour(),
	}
	d := t.Format(time.DateOnly)
	if s.lastMoment == m && s.lastDate == d {
		return []string{}
	}
	s.lastMoment = m
	s.lastDate = d
	return s.moments[m]
}

// Scraping contains config options that apply to One Newsletter's scraping
// behavior
type Scraping struct {
	Schedule       NotificationSchedule
	StorageDirPath string
	// Run the scraper once, then exit
	OneOff bool
	// Print the HTML body of a single email to stdout and exit to help test
	// configuration.
	TestMode bool
	// Number of days we keep a link in the database before marking it
	// expired.
	LinkExpiryDays uint
}

// CheckAndSetDefaults validates s and either returns a copy of s with default
// settings applied or returns an error due to an invalid configuration
func (s *Scraping) CheckAndSetDefaults() (Scraping, error) {
	if s.StorageDirPath == "" {
		return Scraping{}, errors.New(
			"user-provided config does not include a storage path",
		)
	}
	if s.LinkExpiryDays == 0 {
		s.LinkExpiryDays = 180
	}

	return *s, nil
}

// UnmarshalYAML parses a user-provided YAML configuration, returning any
// parsing errors.
func (s *Scraping) UnmarshalYAML(unmarshal func(interface{}) error) error {
	v := make(map[string]string)
	err := unmarshal(&v)
	if err != nil {
		return fmt.Errorf("can't parse the user config: %v", err)
	}

	sp, ok := v["storageDir"]
	if !ok {
		sp = ""
	}

	s.StorageDirPath = sp

	li, ok := v["linkExpiryDays"]
	if !ok {
		li = "0"
	}

	lid, err := strconv.Atoi(li)
	if err != nil {
		return errors.New("can't parse the link expiry as an integer")
	}
	s.LinkExpiryDays = uint(lid)

	ni, ok := v["schedule"]
	if !ok {
		return errors.New("the configuration must provide a notification schedule")
	}

	n, err := parseNotificationSchedule(ni)
	if err != nil {
		return fmt.Errorf("cannot parse the notification schedule: %w", err)
	}
	s.Schedule = n

	return nil
}

// CheckAndSetDefaults validates m and either returns a copy of m with default
// settings applied or returns an error due to an invalid configuration
func (m *Meta) CheckAndSetDefaults() (Meta, error) {
	c := Meta{}

	s, err := m.Scraping.CheckAndSetDefaults()
	if err != nil {
		return Meta{}, err
	}
	c.Scraping = s

	e, err := m.EmailSettings.CheckAndSetDefaults()
	if err != nil {
		return Meta{}, err
	}
	c.EmailSettings = e

	c.LinkSources = make([]linksrc.Config, len(m.LinkSources))
	for n, s := range m.LinkSources {
		ns, err := s.CheckAndSetDefaults()
		if err != nil {
			return Meta{}, err
		}
		c.LinkSources[n] = ns
	}

	return c, nil

}

// Parse generates usable configurations from possibly arbitrary user input.
// An error indicates a problem with parsing or validation. The Reader r
// can be either JSON or YAML.
func Parse(r io.Reader) (*Meta, error) {
	var m Meta
	err := yaml.NewDecoder(r).Decode(&m)
	if err != nil {
		return &Meta{}, fmt.Errorf("can't read the config file as YAML: %v", err)
	}

	var es email.UserConfig = email.UserConfig{}
	if m.EmailSettings == es {
		return &Meta{}, errors.New("must include an \"email\" section")
	}

	var sc Scraping = Scraping{}
	if m.Scraping == sc {
		return &Meta{}, errors.New("must include a \"scraping\" section")
	}

	if len(m.LinkSources) == 0 {
		return &Meta{}, errors.New("must include at least one item within \"link_sources\"")
	}

	// Since this is a one-off or a test, set the data directory to an
	// empty string to disable database operations.
	if m.Scraping.OneOff || m.Scraping.TestMode {
		m.Scraping.StorageDirPath = ""
		log.Debug().Msg(
			"disabling database operations",
		)
	}

	return &m, nil

}

var dayMarkerPattern = regexp.MustCompile(`[A-Z][a-z]?`)

// parseNotificationSchedule creates a NotificationSchedule based on the value
// in val. It returns an error if the value is invalid.
func parseNotificationSchedule(val string) (NotificationSchedule, error) {
	parts := strings.Split(val, " ")
	if len(parts) != 2 {
		return NotificationSchedule{}, errors.New(`notification schedules must include days and an hour, separated by a space, such as "MWF 12"`)
	}

	ret := NotificationSchedule{}
	days := dayMarkerPattern.FindAllString(parts[0], -1)
	for _, d := range days {
		switch d {
		case "M":
			ret.Weekdays |= Monday
		case "Tu":
			ret.Weekdays |= Tuesday
		case "W":
			ret.Weekdays |= Wednesday
		case "Th":
			ret.Weekdays |= Thursday
		case "F":
			ret.Weekdays |= Friday
		case "Sa":
			ret.Weekdays |= Saturday
		case "Su":
			ret.Weekdays |= Sunday
		default:
			return NotificationSchedule{}, fmt.Errorf("not a valid notification schedule day: %q", d)
		}
	}
	h, err := strconv.Atoi(parts[1])
	if err != nil {
		return NotificationSchedule{}, fmt.Errorf("cannot convert %v into an hour while parsing a notification schedule: cannot be converted into an integer - did you swap the hour and days?", parts[1])
	}
	ret.Hour = h
	return ret, nil
}
