package server

import "html/template"

var (
	ruleUpdateBannerTmpl = template.Must(template.New("ruleBanner").Parse(
		`<div id="rule-banner" class="alert alert-success position-fixed top-0 start-50 translate-middle-x mt-3" style="z-index:2000; min-width:300px; text-align:center;">Updated {{.Count}} transactions</div><script>setTimeout(function(){ var b=document.getElementById('rule-banner'); if(b){b.remove();}}, 3500);</script>`))

	payeeEditFormTmpl = template.Must(template.New("payeeEdit").Parse(
		`<form hx-patch="/transactions/{{.ID}}" hx-trigger="blur from:input, submit" hx-target="this" hx-swap="outerHTML" style="display:inline;">
<input type="text" name="payee" value="{{.Payee}}" class="form-control form-control-sm w-auto d-inline" autofocus onblur="this.form.requestSubmit()">
</form>`))
)
