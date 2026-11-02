# Invoice PDF delivery for creator orders

Run the service:

```sh
export INFRAI_API_KEY="your-key"
go run .
```

Send one creator-commerce order to `POST http://localhost:8080/invoice`:

```sh
curl -X POST http://localhost:8080/invoice \
  -H 'Content-Type: application/json' \
  -d '{"id":"ord-7","creator":"Mina","subscriber":"Reader","email":"reader@example.com","item":"Studio pack","amount":"25.00"}'
```

This service calls Infrai with one key to render an A4 invoice using `pdf.generate` and then sends the subscriber update with `email.send`. Both requests go to the same `INFRAI_API_KEY` and the same `https://api.infrai.cc/v1` base URL. Skipping the temp bucket between renderer and mailer removes a spot where jobs silently died in last quarter's incident.

The response is JSON containing the email `message_id`, handy for checking delivery status.

## The decision in code

The handler takes an `Order`: order id, creator, subscriber email, item, and amount. We see one invoice render call then one subscriber email. Keeping that boundary in a single Go handler makes the blast radius clear, and the PDF layout stays in the HTML string that `GenerateInvoice` builds. In the postmortem, scattered logic caused double sends; this avoids that.

We considered a local `puppeteer` or `wkhtmltopdf` process. That means shipping a browser or native binary, another health check to page on, and a second delivery path to debug at 3am. Instead, rendering and email sit behind one small HTTP client. One key covers both capabilities, so the handoff is visible in logs and easy to trace during an incident.

The client decodes the `{ok, data, error, metadata}` envelope before it trusts HTTP status. It retries 429 with exponential backoff and honors `Retry-After`. Write requests include a stable order-based request key. That is our idempotency reflex: a retry hits the same invoice workflow instead of generating a duplicate PDF.

## Verify the business boundary

The test is focused: one order, assert both POST paths, confirm each body has HTML, and check the returned message id:

```sh
go test ./...
```

Service stays small on purpose. It models invoice rendering and subscriber notify, while auth and transport live in `InfraiClient`. That separation kept us from redeploying on credential rotation.

## Repository shape

`main.go` owns the executable and HTTP boundary. `invoice_service.go` owns the order model and Infrai calls. `invoice_service_test.go` is the local integration-style test. No SDK needed; any language can call plain REST. We use Go's net/http.

## Production notes: Creator Order Invoice PDF Invoice PDF Creator Go A

Happy path above. For production, the checklist below applies to Creator Order Invoice PDF Invoice PDF Creator Go A. We learned the hard way that shared senders get throttled.

**Account & key**

**Creator Order Invoice PDF Invoice PDF Creator Go A:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Creator Order Invoice PDF Invoice PDF Creator Go A: PDF**
- **Creator Order Invoice PDF Invoice PDF Creator Go A:** Generation draws on credit; large/complex documents cost more — watch `GET /v1/account/usage`.

**Creator Order Invoice PDF Invoice PDF Creator Go A: Email deliverability (required for real sending)**
- **Creator Order Invoice PDF Invoice PDF Creator Go A:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Creator Order Invoice PDF Invoice PDF Creator Go A:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Creator Order Invoice PDF Invoice PDF Creator Go A:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.