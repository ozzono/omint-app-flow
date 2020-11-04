package omint

import (
	"fmt"
	"testing"
)

func TestExpressions(t *testing.T) {
	expressions := allExpressions()
	defaultCoordExp := "\\(\\\\\\[\\\\d\\+,\\\\d\\+\\\\\\]\\\\\\[\\\\d\\+,\\\\d\\+\\\\\\]\\)"
	testExpressions := map[string]string{
		"menu-btn":     fmt.Sprintf("(logo-omint-letters\\.\\*\\?%s)", defaultCoordExp),                                                                                              //'logo-omint-letters.*?%s'
		"invoice-btn":  fmt.Sprintf("(Faturas\\.\\*\\?%s)", defaultCoordExp),                                                                                                         //'Faturas.*?%s'
		"invoice-pdf":  fmt.Sprintf("(%s\" /><node index=\"\\.\" text=\"N.. \\\\d\\{7\\}\")", defaultCoordExp),                                                                       //'%s" /><node index="." text="N°: \d{7}"'
		"ok-btn":       fmt.Sprintf("(OK\\.\\*\\?%s)", defaultCoordExp),                                                                                                              //'OK.*?%s'
		"more-options": fmt.Sprintf("(More options\\.\\*\\?%s)", defaultCoordExp),                                                                                                    //'More options.*?%s'
		"dl-btn":       fmt.Sprintf("(Download\\.\\*\\?%s)", defaultCoordExp),                                                                                                        //'Download.*?%s'
		"login-btn":    fmt.Sprintf("(loginr\\.\\*\\?%s)", defaultCoordExp),                                                                                                          // "loginr.*?(\\[\\d+,\\d+\\]\\[\\d+,\\d+\\])",
		"access-btn":   fmt.Sprintf("(Acessar\\.\\*\\?%s)", defaultCoordExp),                                                                                                         // "Acessar.*?(\\[\\d+,\\d+\\]\\[\\d+,\\d+\\])",
		"deny-btn":     fmt.Sprintf("(DENY\\.\\*\\?%s)", defaultCoordExp),                                                                                                            // "DENY.*?(\\[\\d+,\\d+\\]\\[\\d+,\\d+\\])",
		"allow-btn":    fmt.Sprintf("(allow_button\\.\\*\\?%s)", defaultCoordExp),                                                                                                    // "allow_button.*?(\\[\\d+,\\d+\\]\\[\\d+,\\d+\\])",
		"pdf-barcode":  fmt.Sprintf("SANTANDER\\.\\(\\\\d\\{5\\}\\\\\\.\\\\d\\{5\\}\\.\\\\d\\{5\\}\\\\\\.\\\\d\\{6\\}\\.\\\\d\\{5\\}\\\\\\.\\\\d\\{6\\}\\.\\\\d\\.\\\\d\\{14\\}\\)"), // SANTANDER.(\d{5}\.\d{5}.\d{5}\.\d{6}.\d{5}\.\d{6}.\d.\d{14})
		"pdf-duedate":  fmt.Sprintf("VENCIMENTO\\.\\(\\\\d\\{2\\}/\\\\d\\{2\\}/20\\\\d\\{2\\}\\)"),                                                                                   // VENCIMENTO.(\d{2}/\d{2}/20\d{2})
		"pdf-value":    fmt.Sprintf("VALOR\\.\\(\\\\d\\+\\\\\\.\\\\d\\{1,3\\},\\\\d\\{1,2\\}\\)"),                                                                                    // "VALOR.(\\d+\\.\\d{1,3},\\d{1,2})",
	}
	for key := range testExpressions {
		if match(testExpressions[key], expressions[key]) {
			t.Logf("matched %s \n", key)
		} else {
			t.Fatalf("failed:\ntestExp: %s\nkey: %s\nexp: %s", testExpressions[key], key, expressions[key])
		}
	}
}
