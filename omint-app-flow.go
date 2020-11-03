package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"

	"code.sajari.com/docconv"
	adb "github.com/ozzono/adbtools"
)

var (
	device      adb.Device
	expressions map[string]string
	login       loginData
	globalLog   bool
	pullFile    string

	omint = app{
		pkg:      "br.com.omint.apps.minhaomint",
		activity: "br.com.omint.apps.minhaomint.MainActivity",
	}
	coordExpression = "\\[\\d+,\\d+\\]\\[\\d+,\\d+\\]"
)

const (
	defaultSleep = 100
)

type flow struct {
	device   adb.Device
	Invoices []Invoice
	Login    loginData
	close    func()
}

type app struct {
	pkg      string
	activity string
}

type loginData struct {
	email string
	pw    string
}

// Invoice has all the payment data
type Invoice struct {
	DueDate string
	Value   string
	BarCode string
	Status  string
}

func init() {
	flag.StringVar(&login.email, "email", "", "Sets the user login email")
	flag.StringVar(&login.pw, "pw", "", "Sets the user login password")
	flag.BoolVar(&globalLog, "log", true, "Sets the global log lvl")
}

func main() {
	flag.Parse()

	expressions = allExpressions()
	if err := checkLoginData(); err != nil {
		log.Printf("checkLoginData err: %v", err)
		return
	}

	flow, err := newFlow()
	if err != nil {
		log.Println(err)
		return
	}
	defer flow.close()

	if !strings.Contains(flow.device.ID, "emulator") {
		flow.device.WakeUp()
		flow.device.Swipe([4]int{int(flow.device.Screen.Width / 2), flow.device.Screen.Height - 100, int(flow.device.Screen.Width / 2), 100})
	}

	invoice, err := flow.OmintInvoice()
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("invoice %#v\n", invoice)
}

func (flow *flow) OmintInvoice() (Invoice, error) {
	deferMe, err := flow.device.ScreenTimeout("10m")
	defer deferMe()
	defer device.ScreenCap("omint.png")
	if err != nil {
		return Invoice{}, fmt.Errorf("ScreenTimeout err: %v", nil)
	}
	flow.device.CloseApp(omint.pkg)
	err = flow.device.StartApp(omint.pkg, omint.activity, "")
	if err != nil {
		return Invoice{}, err
	}

	flow.sleep(50)

	if !strings.Contains(flow.device.Foreground(), omint.pkg) {
		xmlScreen, err := device.XMLScreen(true)
		if err != nil {
			return Invoice{}, err
		}
		if match("Allow.*to access this device", xmlScreen) {
			coords, err := adb.XMLtoCoords(applyRegexp(expressions["deny-btn"], xmlScreen)[1])
			if err != nil {
				return Invoice{}, err
			}
			log.Println("Denying access to device's location")
			flow.device.TapScreen(coords[0], coords[1], 10)
		}
	}

	if !flow.device.WaitApp(omint.pkg, 1000, 10) {
		return Invoice{}, fmt.Errorf("%s app did not start", omint.pkg)
	}

	flow.sleep(50)

	if flow.device.HasInScreen(true, "Primeiro acesso", "login") {
		if err := flow.loginFlow(); err != nil {
			return Invoice{}, fmt.Errorf("loginFlow err: %v", err)
		}
	}

	flow.sleep(20)

	nonStop, err := flow.invoicePDF()
	if err != nil {
		return Invoice{}, err
	}

	if !nonStop {
		return Invoice{}, fmt.Errorf("No pending invoice available")
	}

	invoice, err := pdfFlow()
	if err != nil {
		return Invoice{}, err
	}

	return invoice, nil
}

func pdfFlow() (Invoice, error) {
	res, err := docconv.ConvertPath("invoice.pdf")
	if err != nil {
		return Invoice{}, fmt.Errorf("docconv.ConvertPath")
	}

	pdfText := fmt.Sprintf("%v", res)
	pdfText = strings.Replace(pdfText, "\n\n", "\n", -1)
	pdfText = strings.Replace(pdfText, "VALOR\n", "VALOR ", -1)
	pdfText = strings.Replace(pdfText, "VENCIMENTO\n", "VENCIMENTO ", 1)
	pdfText = strings.Replace(pdfText, "SANTANDER\n", "SANTANDER ", -1)

	err = os.Remove("invoice.pdf")
	if err != nil {
		log.Printf("failed to remove invoice.pdf file; this must be verified")
	}

	return Invoice{
		Value:   applyRegexp(expressions["pdf-value"], pdfText)[1],
		DueDate: applyRegexp(expressions["pdf-duedate"], pdfText)[1],
		BarCode: barCode(applyRegexp(expressions["pdf-barcode"], pdfText)[1]),
		Status:  "pending",
	}, nil
}

func barCode(input string) string {
	input = strings.Replace(input, " ", "", -1)
	input = strings.Replace(input, ".", "", -1)
	return input
}

func (flow *flow) invoicePDF() (bool, error) {
	log.Println("Starting invoice PDF flow")

	err := flow.exp2tap(expressions["menu-btn"])
	if err != nil {
		return false, err
	}

	flow.sleep(10)

	err = flow.exp2tap(expressions["invoice-btn"])
	if err != nil {
		return false, err
	}

	err = flow.device.WaitInScreen(5, "vencimento", "pagamento")
	if err != nil {
		return false, err
	}
	if !flow.device.HasInScreen(true, "aberto") {
		return false, nil
	}

	err = flow.exp2tap(expressions["invoice-pdf"])
	if err != nil {
		return false, err
	}

	err = device.WaitInScreen(10, "fatura", "pdf")
	if err != nil {
		return false, err
	}
	err = device.WaitInScreen(10, "download", "ok", "cancel")
	if err != nil {
		return false, err
	}

	err = flow.exp2tap(expressions["ok-btn"])
	if err != nil {
		return false, err
	}

	err = flow.exp2tap(expressions["more-options"])
	if err != nil {
		return false, err
	}

	path, err := flow.storagePath()
	if err != nil {
		return false, err
	}
	download := fmt.Sprintf("%s/Download", path)
	flow.device.Shell(fmt.Sprintf("adb shell rm %s/*", download))

	err = flow.exp2tap(expressions["dl-button"])
	if err != nil {
		return false, err
	}

	if err := flow.device.WaitInScreen(2, "allow_button", "deny", "permission_message"); err != nil {
		log.Printf("Permission already given; err: %v", nil)
	} else {
		err = flow.exp2tap(expressions["allow-button"])
		if err != nil {
			return false, err
		}
	}

	file := applyRegexp(
		"(Fatura_\\d{7}.pdf)", //Fatura_0000000.pdf
		flow.device.Shell("adb shell ls "+download))[1]
	if len(file) == 0 {
		return false, fmt.Errorf("failed to find invoice pdf file")
	}
	if !strings.Contains(flow.device.Shell(fmt.Sprintf("adb pull %s/%s invoice.pdf", download, file)), "1 file pulled") {
		return false, fmt.Errorf("failed to pull %s/%s", download, file)
	}

	flow.device.Shell(fmt.Sprintf("adb shell rm %s/*", download))

	return true, nil
}

func (flow *flow) loginFlow() error {
	log.Println("Starting login flow")

	err := flow.exp2tap(expressions["login-btn"])
	if err != nil {
		return err
	}

	nodes := [][2]int{}
	for _, item := range device.NodeList(true) {
		if match(coordExpression, item) && strings.Contains(item, "NAF") {
			coords, err := adb.XMLtoCoords(applyRegexp(fmt.Sprintf("(%s)", coordExpression), item)[1])
			if err != nil {
				return err
			}
			nodes = append(nodes, coords)
		}
	}

	if len(nodes) != 2 {
		return fmt.Errorf("failed to fetch login and pw nodes; nodes found: %v", nodes)
	}

	// email input
	flow.device.TapScreen(nodes[0][0], nodes[0][1], 10)
	flow.sleep(5)
	flow.device.InputText(flow.Login.email, false)

	// pw input
	flow.device.TapScreen(nodes[1][0], nodes[1][1], 10)
	flow.sleep(5)
	flow.device.InputText(flow.Login.pw, false)

	err = flow.exp2tap(expressions["access-btn"])
	if err != nil {
		return err
	}

	flow.sleep(50)

	err = flow.device.WaitInScreen(5, "credencial", "contatos", "atendimento")
	if err != nil {
		return err
	}

	log.Println("Successfully logged in")
	return nil
}

func applyRegexp(exp, text string) []string {
	re := regexp.MustCompile(exp)
	matches := re.FindStringSubmatch(text)
	if len(matches) < 1 {
		fmt.Printf("Unable to find match for exp %s\n", exp)
		return []string{}
	}
	return matches
}

func newRandInt(i int) int {
	return rand.New(rand.NewSource(time.Now().UnixNano())).Intn(i)
}

func allExpressions() map[string]string {
	//default expression (\[\d+,\d+\]\[\d+,\d+\])
	return map[string]string{
		"login-btn":    "loginr.*?(\\[\\d+,\\d+\\]\\[\\d+,\\d+\\])",
		"access-btn":   "Acessar.*?(\\[\\d+,\\d+\\]\\[\\d+,\\d+\\])",
		"deny-btn":     "DENY.*?(\\[\\d+,\\d+\\]\\[\\d+,\\d+\\])",
		"menu-btn":     "logo-omint-letters.*?(\\[\\d+,\\d+\\]\\[\\d+,\\d+\\])",
		"invoice-btn":  "Faturas.*?(\\[\\d+,\\d+\\]\\[\\d+,\\d+\\])",
		"invoice-pdf":  "(\\[\\d+,\\d+\\]\\[\\d+,\\d+\\])\" /><node index=\".\" text=\"N°: \\d{7}\"",
		"ok-btn":       "OK.*?(\\[\\d+,\\d+\\]\\[\\d+,\\d+\\])",
		"more-options": "More options.*?(\\[\\d+,\\d+\\]\\[\\d+,\\d+\\])",
		"allow-button": "allow_button.*?(\\[\\d+,\\d+\\]\\[\\d+,\\d+\\])",
		"dl-button":    "Download.*?(\\[\\d+,\\d+\\]\\[\\d+,\\d+\\])",
		"pdf-value":    "VALOR.(\\d+\\.\\d{1,3},\\d{1,2})",
		"pdf-duedate":  "VENCIMENTO.(\\d{2}/\\d{2}/20\\d{2})",
		"pdf-barcode":  "SANTANDER.(\\d{5}\\.\\d{5}.\\d{5}\\.\\d{6}.\\d{5}\\.\\d{6}.\\d.\\d{14})",
	}
}

func checkLoginData() error {
	log.Printf("Checking login data: %v", login)
	if len(login.email) == 0 || !strings.Contains(login.email, "@") {
		return fmt.Errorf("Invalid user email")
	}
	if len(login.pw) == 0 {
		return fmt.Errorf("Invalid user password")
	}
	return nil
}

func (flow *flow) defaultSleep(delay int) {
	flow.device.DefaultSleep = delay
}

func match(exp, text string) bool {
	return regexp.MustCompile(exp).MatchString(text)
}

func newFlow() (flow, error) {
	device, has := hasEmulator()
	if has {
		return flow{
			device: device,
			close:  func() {},
			Login:  login,
		}, nil
	}

	close, err := adb.StartAVD(true, "lite")
	if err != nil {
		close()
		return flow{}, err
	}
	time.Sleep(5 * time.Second)
	devices, err := adb.Devices(globalLog)
	if err != nil {
		return flow{}, err
	}
	return flow{
		device: devices[0],
		close:  close,
		Login:  login,
	}, nil

}

func hasEmulator() (adb.Device, bool) {
	devices, err := adb.Devices(true)
	if err != nil {
		return adb.Device{}, false
	}
	if len(devices) == 0 {
		return adb.Device{}, false
	}
	for i := range devices {
		if strings.HasSuffix(devices[i].ID, "emulator") {
			return devices[i], true
		}
	}
	return adb.Device{}, false
}

func waitEnter() {
	log.Printf("Press <enter> to continue or <ctrl+c> to interrupt")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	log.Printf("Now, where was I?")
	log.Printf("Oh yes...\n\n")
}

func (flow *flow) sleep(delay int) {
	time.Sleep(time.Duration(delay*flow.device.DefaultSleep) * time.Millisecond)
}

func (flow *flow) exp2tap(exp string) error {
	xmlScreen, err := device.XMLScreen(true)
	if err != nil {
		return err
	}
	coords, err := adb.XMLtoCoords(applyRegexp(exp, xmlScreen)[1])
	if err != nil {
		return err
	}
	flow.device.TapScreen(coords[0], coords[1], 10)
	return nil
}

func (flow *flow) storagePath() (string, error) {
	dumpOutput := flow.device.Shell("adb shell uiautomator dump")
	if !strings.Contains(dumpOutput, "dump.xml") {
		return "", fmt.Errorf("Failed to dump screen xml: %s", dumpOutput)
	}
	dumpOutput = strings.Replace(dumpOutput, "\n", "", -1)
	dumpOutput = strings.TrimPrefix(dumpOutput, "UI hierchary dumped to: ")
	dumpOutput = strings.TrimSuffix(dumpOutput, "/window_dump.xml")
	return dumpOutput, nil
}
