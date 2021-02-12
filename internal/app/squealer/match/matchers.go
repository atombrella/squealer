package match

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing/object"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strings"

	"github.com/owenrumney/squealer/internal/app/squealer/config"
	"github.com/owenrumney/squealer/internal/app/squealer/mertics"
)

type Matcher struct {
	test        *regexp.Regexp
	description string
}

type Matchers []*Matcher

type MatcherController struct {
	matchers       Matchers
	exclusions     []config.RuleException
	transgressions *transgressionMap
	metrics        *mertics.Metrics
	redacted       bool
}

func NewMatcherController(cfg *config.Config, metrics *mertics.Metrics, redacted bool) *MatcherController {
	mc := &MatcherController{
		matchers:       []*Matcher{},
		transgressions: newTransgressions(),
		exclusions:     cfg.Exceptions,
		metrics:        metrics,
		redacted:       redacted,
	}

	for _, rule := range cfg.Rules {
		err := mc.add(rule)
		if err != nil {
			log.WithError(err).Error(err.Error())
		}
	}

	return mc
}

func (mc *MatcherController) add(rule config.MatchRule) error {
	compile, err := regexp.Compile(rule.Rule)
	if err != nil {
		return fmt.Errorf("failed to compile the regex. %v", err.Error())
	}
	mc.matchers = append(mc.matchers, &Matcher{
		test:        compile,
		description: rule.Description,
	})
	return nil
}

func (mc *MatcherController) Evaluate(filename, content string, commit *object.Commit) error {
	log.Debugf("\tfile: %s", filename)
	for _, matcher := range mc.matchers {
		if matcher.test.MatchString(content) {
			mc.addTransgression(&content, filename, matcher, commit)
		}
	}
	return nil
}

func (mc *MatcherController) addTransgression(content *string, name string, matcher *Matcher, commit *object.Commit) {
	lines := strings.Split(*content, "\n")

	m := matcher.test.FindString(*content)
	if len(m) > 0 {
		lineContent := lineInFile(m, lines)
		secretHash := mc.newHash(m)
		key := fmt.Sprintf("%s:%s", name, secretHash)
		mc.metrics.IncrementTransgressionsFound()
		for _, exclusion := range mc.exclusions {
			if exclusion.ExceptionString == key {
				mc.metrics.IncrementTransgressionsIgnored()
				return
			}
		}

		if !mc.transgressions.exists(key) {
			mc.metrics.IncrementTransgressionsReported()
			transgression := newTransgression(lineContent, name, m, secretHash, commit)
			mc.transgressions.add(key, transgression)
			log.Debugf("recording transgression in commit: %s", transgression.commitHash)
		}
	}
}

func (mc *MatcherController) newHash(secret string) string {
	hasher := sha1.New()
	hasher.Write([]byte(secret))
	hash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
	return hash
}

func (mc *MatcherController) Transgressions() []Transgression {
	var transgressions []Transgression

	for _, t := range mc.transgressions.internal {
		transgressions = append(transgressions, *t)
	}
	return transgressions
}

func lineInFile(m string, lines []string) string {
	for _, line := range lines {
		if strings.Contains(line, m) {
			return line
		}
	}
	return ""
}
