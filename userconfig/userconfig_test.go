package userconfig

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"gopkg.in/yaml.v2"
)

func TestParse(t *testing.T) {
	// Asserting deep equality between the expected and actual Meta would
	// be really convoluted and brittle, so we should make sure nothing
	// fails unexpectedly and test knottier marshaling/validation situations
	// elswhere.
	testCases := []struct {
		description   string
		conf          string
		shouldBeError bool
		shouldBeEmpty bool
	}{
		{
			description:   "valid case",
			shouldBeError: false,
			shouldBeEmpty: false,
			conf: `---
email:
    smtpServerAddress: smtp://0.0.0.0:123
    fromAddress: mynewsletter@example.com
    toAddress: recipient@example.com
    username: MyUser123
    password: 123456-A_BCDE
link_sources:
    - name: site-38911
      url: http://127.0.0.1:38911
      itemSelector: "ul li"
      captionSelector: "p"
      linkSelector: "a"
scraping:
    schedule: "M 12"
    storageDir: ./tempTestDir3012705204`,
		},
		{
			description:   "no email section",
			shouldBeError: true,
			shouldBeEmpty: true,
			conf: `---
link_sources:
    - name: site-38911
      url: http://127.0.0.1:38911
      itemSelector: "ul li"
      captionSelector: "p"
      linkSelector: "a"
scraping:
    interval: 5s
    storageDir: ./tempTestDir3012705204`,
		},
		{
			description:   "no link_sources section",
			shouldBeError: true,
			shouldBeEmpty: true,
			conf: `---
email:
    smtpServerAddress: smtp://0.0.0.0:123
    fromAddress: mynewsletter@example.com
    toAddress: recipient@example.com
    username: MyUser123
    password: 123456-A_BCDE
scraping:
    interval: 5s
    storageDir: ./tempTestDir3012705204`,
		},
		{
			description:   "no scraping section",
			shouldBeError: true,
			shouldBeEmpty: true,
			conf: `---
email:
    smtpServerAddress: smtp://0.0.0.0:123
    fromAddress: mynewsletter@example.com
    toAddress: recipient@example.com
    username: MyUser123
    password: 123456-A_BCDE
link_sources:
    - name: site-38911
      url: http://127.0.0.1:38911
      itemSelector: "ul li"
      captionSelector: "p"
      linkSelector: "a"`,
		},
		{
			description:   "not yaml",
			shouldBeError: true,
			shouldBeEmpty: true,
			conf:          `this is not yaml`,
		},
		{
			description:   "no item selector or caption selector",
			shouldBeError: false,
			shouldBeEmpty: false,
			conf: `---
email:
    smtpServerAddress: smtp://0.0.0.0:123
    fromAddress: mynewsletter@example.com
    toAddress: recipient@example.com
    username: MyUser123
    password: 123456-A_BCDE
link_sources:
    - name: site-38911
      url: http://127.0.0.1:38911
      linkSelector: "a"
scraping:
    schedule: "M 12"
    storageDir: ./tempTestDir3012705204`,
		},
		{
			description:   "valid link source with no link selector",
			shouldBeError: false,
			shouldBeEmpty: false,
			conf: `---
email:
    smtpServerAddress: smtp://0.0.0.0:123
    fromAddress: mynewsletter@example.com
    toAddress: recipient@example.com
    username: MyUser123
    password: 123456-A_BCDE
link_sources:
    - name: site-38911
      url: http://127.0.0.1:38911
scraping:
    schedule: "M 13"
    storageDir: ./tempTestDir3012705204`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			b := bytes.NewBuffer([]byte(tc.conf))
			m, err := Parse(b)

			if (err != nil) != tc.shouldBeError {
				t.Errorf(
					"%v: unexpected error status: wanted %v but got %v with error %v",
					tc.description,
					tc.shouldBeError,
					err != nil,
					err,
				)
			}

			if reflect.DeepEqual(*m, Meta{}) != tc.shouldBeEmpty {
				l := map[bool]string{
					true:  "to be",
					false: "not to be",
				}
				t.Errorf(
					"%v: expected the Meta %v nil, but got the opposite",
					tc.description,
					l[tc.shouldBeEmpty],
				)
			}
		})

	}

}

func TestScrapingUnmarshalYAML(t *testing.T) {
	testCases := []struct {
		description   string
		input         string
		shouldBeError bool
		expected      Scraping
	}{
		{
			description:   "valid case",
			shouldBeError: false,
			input: `storageDir: ./tempTestDir3012705204
linkExpiryDays: 100
`,
			expected: Scraping{
				StorageDirPath: "./tempTestDir3012705204",
				OneOff:         false,
				TestMode:       false,
				LinkExpiryDays: 100,
			},
		},
		{
			description:   "not an object",
			shouldBeError: true,
			input:         `[]`,
			expected:      Scraping{},
		},
		{
			description:   "unparseable duration",
			shouldBeError: true,
			input: `interval: 5y
storageDir: ./tempTestDir3012705204`,
			expected: Scraping{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			var s Scraping
			empty := Scraping{}
			if err := yaml.NewDecoder(
				bytes.NewBuffer([]byte(tc.input)),
			).Decode(&s); (err != nil) != tc.shouldBeError {
				t.Errorf(
					"expected error status to be %v but got error %v",
					tc.shouldBeError,
					err,
				)
			}
			if tc.expected != empty {
				assert.Equal(t, tc.expected, s)
			}
		})
	}
}

func TestScrapingCheckAndSetDefaults(t *testing.T) {
	cases := []struct {
		description        string
		input              Scraping
		expected           Scraping
		expectErrSubstring string
	}{

		{
			description: "no storage path",
			input: Scraping{
				OneOff:   false,
				TestMode: false,
			},
			expected:           Scraping{},
			expectErrSubstring: "path",
		},
		{
			description: "valid config with no link TTL",
			input: Scraping{
				OneOff:         false,
				TestMode:       false,
				StorageDirPath: "/storage",
			},
			expected: Scraping{
				StorageDirPath: "/storage",
				OneOff:         false,
				TestMode:       false,
				LinkExpiryDays: 180,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			actual, err := c.input.CheckAndSetDefaults()
			if c.expectErrSubstring != "" && err == nil {
				t.Fatalf(
					"expected an error with substring %v but got nil",
					c.expectErrSubstring,
				)
			}
			if c.expectErrSubstring != "" &&
				!strings.Contains(err.Error(), c.expectErrSubstring) {
				t.Fatalf(
					"expected error with substring %v but got %v",
					c.expectErrSubstring,
					err,
				)
			}
			if c.expectErrSubstring == "" && err != nil {
				t.Fatalf("expected no error but got %v", err)
			}
			if !reflect.DeepEqual(actual, c.expected) {
				t.Fatalf("expected %+v but got %+v", c.expected, actual)
			}
		})
	}
}

func Test_moments(t *testing.T) {
	cases := []struct {
		description string
		input       NotificationSchedule
		expected    []Moment
	}{
		{
			description: "wednesdays at noon",
			input: NotificationSchedule{
				Weekdays: Wednesday,
				Hour:     12,
			},
			expected: []Moment{
				{
					Day:  time.Wednesday,
					Hour: 12,
				},
			},
		},
		{
			description: "mondays and fridays at 1pm",
			input: NotificationSchedule{
				Weekdays: Friday | Monday,
				Hour:     13,
			},
			expected: []Moment{
				{
					Day:  time.Monday,
					Hour: 13,
				},
				{
					Day:  time.Friday,
					Hour: 13,
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			assert.Equal(t, c.expected, moments(c.input))
		})
	}
}

func TestAddGet(t *testing.T) {
	cases := []struct {
		description string
		input       NotificationSchedule
		currentTime string // DateTime format
	}{
		{
			description: "wednesdays at noon",
			input: NotificationSchedule{
				Weekdays: Wednesday,
				Hour:     12,
			},
			currentTime: "2025-06-04 12:00:00",
		},
		{
			description: "mondays and fridays at 1pm",
			input: NotificationSchedule{
				Weekdays: Friday | Monday,
				Hour:     13,
			},
			currentTime: "2025-06-02 13:00:00",
		},
	}

	for _, c := range cases {
		s := NewScheduleStore()
		s.Add("myhandle", c.input)
		m, err := time.Parse(time.DateTime, c.currentTime)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t,
			[]string{"myhandle"}, s.Get(m),
		)
	}
}

func TestGet(t *testing.T) {
	moments := map[string]NotificationSchedule{
		"moment1": {
			Weekdays: Monday,
			Hour:     12,
		},
		"moment2": {
			Weekdays: Tuesday,
			Hour:     13,
		},
		"moment3": {
			Weekdays: Wednesday,
			Hour:     14,
		},
		"moment4": {
			Weekdays: Wednesday,
			Hour:     14,
		},
	}

	t.Run("multiple matches", func(t *testing.T) {
		s := NewScheduleStore()
		for k, v := range moments {
			s.Add(k, v)
		}
		expected := []string{"moment3", "moment4"}
		m, err := time.Parse(time.DateTime, "2025-06-04 14:00:00")
		if err != nil {
			t.Fatal(err)
		}
		actual := s.Get(m)

		assert.Equal(t, expected, actual)
	})

	t.Run("successive identical gets", func(t *testing.T) {
		s := NewScheduleStore()
		for k, v := range moments {
			s.Add(k, v)
		}
		expected := []string{}
		m1, err := time.Parse(time.DateTime, "2025-06-04 12:00:00")
		if err != nil {
			t.Fatal(err)
		}
		m2, err := time.Parse(time.DateTime, "2025-06-04 12:01:00")
		if err != nil {
			t.Fatal(err)
		}

		// Ignoring the result of the first Get.
		s.Get(m1)
		actual := s.Get(m2)

		assert.Equal(t, expected, actual)
	})

	t.Run("identical gets in successive weeks", func(t *testing.T) {
		s := NewScheduleStore()
		for k, v := range moments {
			s.Add(k, v)
		}
		expected := []string{"moment1"}
		m1, err := time.Parse(time.DateTime, "2025-06-02 12:00:00")
		if err != nil {
			t.Fatal(err)
		}
		m2, err := time.Parse(time.DateTime, "2025-06-09 12:00:00")
		if err != nil {
			t.Fatal(err)
		}

		// Ignoring the result of the first Get.
		s.Get(m1)
		actual := s.Get(m2)

		assert.Equal(t, expected, actual)
	})

	t.Run("successive different gets", func(t *testing.T) {
		s := NewScheduleStore()
		for k, v := range moments {
			s.Add(k, v)
		}
		expected := []string{"moment1"}
		m1, err := time.Parse(time.DateTime, "2025-06-02 12:00:00")
		if err != nil {
			t.Fatal(err)
		}
		m2, err := time.Parse(time.DateTime, "2025-06-02 14:00:00")
		if err != nil {
			t.Fatal(err)
		}

		// Ignoring the result of the first Get.
		s.Get(m2)
		actual := s.Get(m1)

		assert.Equal(t, expected, actual)
	})

}

func Test_parseNotificationSchedule(t *testing.T) {
	cases := []struct {
		description string
		input       string
		expected    NotificationSchedule
	}{
		{
			description: "single-day schedule",
			input:       "M 12",
			expected: NotificationSchedule{
				Weekdays: Monday,
				Hour:     12,
			},
		},
		{
			description: "multi-day schedule: days in order",
			input:       "MTuW 13",
			expected: NotificationSchedule{
				Weekdays: Monday | Tuesday | Wednesday,
				Hour:     13,
			},
		},
		{
			description: "multi-day schedule: days out of order",
			input:       "WMTu 13",
			expected: NotificationSchedule{
				Weekdays: Monday | Tuesday | Wednesday,
				Hour:     13,
			},
		},
		{
			description: "repeated days",
			input:       "MM 15",
			expected: NotificationSchedule{
				Weekdays: Monday,
				Hour:     15,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			s, err := parseNotificationSchedule(c.input)
			assert.NoError(t, err)
			assert.Equal(t, c.expected, s)
		})
	}
}

func Test_parseNotificationSchedule_invalid(t *testing.T) {
	cases := []struct {
		description  string
		input        string
		errSubstring string
	}{
		{
			description:  "missing time",
			input:        "M",
			errSubstring: `notification schedules must include days and an hour, separated by a space, such as "MWF 12"`,
		},
		{
			description:  "missing days",
			input:        "12",
			errSubstring: `notification schedules must include days and an hour, separated by a space, such as "MWF 12"`,
		},
		{
			description:  "empty input",
			input:        "",
			errSubstring: `notification schedules must include days and an hour, separated by a space, such as "MWF 12"`,
		},
		{
			description:  "unexpected day",
			input:        "MTW 12",
			errSubstring: `not a valid notification schedule day: "T"`,
		},
		{
			description:  "parts reversed",
			input:        "12 MT",
			errSubstring: `cannot convert MT into an hour while parsing a notification schedule: cannot be converted into an integer - did you swap the hour and days?`,
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			_, err := parseNotificationSchedule(c.input)
			assert.ErrorContains(t, err, c.errSubstring)
		})
	}
}
