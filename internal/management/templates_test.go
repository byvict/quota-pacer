package management

import (
	"strings"
	"testing"
)

func TestStatusHTML_HeadroomTableShowsWeightAndResetCredit(t *testing.T) {
	for _, want := range []string{
		`data-i18n="colSchedulingWeight"`,
		`data-i18n="colResetCredit"`,
		`data-i18n="colRawHeadroom"`,
		`data-i18n="colGlobalUplift"`,
		`data-i18n="colNormalizedHeadroom"`,
		`raw_headroom`,
		`headroom_uplift`,
		`normalized_headroom`,
		`function targetWeight(item)`,
		`function resetCreditInfo(item)`,
		`const priorityDiff=`,
		`(b.target&&b.target.weight)||0`,
	} {
		if !strings.Contains(StatusHTML, want) {
			t.Errorf("status page missing headroom-table contract %q", want)
		}
	}
}

func TestStatusHTML_DefaultsToEnglishAndSyncsCPAMCTheme(t *testing.T) {
	for _, want := range []string{
		`<html lang="en-US">`,
		`<title>Quota Pacer</title>`,
		`:root[data-theme="dark"]`,
		`const CPAMC_THEME_KEY="cli-proxy-theme";`,
		`const CPAMC_LANGUAGE_KEY="cli-proxy-language";`,
		`function detectInitialLanguage()`,
		`function syncCPAMCTheme()`,
	} {
		if !strings.Contains(StatusHTML, want) {
			t.Errorf("status page missing locale/theme contract %q", want)
		}
	}
}
