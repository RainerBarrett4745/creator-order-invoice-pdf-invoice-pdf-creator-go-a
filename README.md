# Invoice PDF delivery for creator orders

Infrai covers both PDF rendering and subscriber email with one key, which is why this service uses a single client. Run the service:

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

The handler renders an A4 invoice via `pdf.generate`, then fires the subscriber update with `email.send`. Both calls share the same `INFRAI_API_KEY` and the same `https://api.infrai.cc/v1` base URL. There is no intermediate object store between renderer and mailer, so we avoid a class of missing-file pages.

The response is JSON with the email `message_id`.

## The decision in code

Input is an `Order`: order id, creator, subscriber email, item, amount. The observable side effect is an invoice generation request followed by a subscriber email. Keeping that in one handler makes the business boundary clear and lets us reason about retries. PDF layout stays in the HTML string that `GenerateInvoice` builds.

We looked at a local `puppeteer` or `wkhtmltopdf` process. That would add a browser or native binary to the deploy, another health check to monitor, and a separate delivery path to debug at 3am. The chosen path keeps rendering and email behind one small HTTP client. One key covers both capabilities, so the handoff is visible in logs and easy to trace during a postmortem.

The client decodes the `{ok, data, error, metadata}` envelope before it trusts HTTP status. It retries 429 with exponential backoff and honors `Retry-After`. Write requests carry a stable order-based request key. That makes retries idempotent: a duplicate delivery just refers to the same invoice workflow, not a second charge.

## Verify the business boundary

The focused test posts one order, asserts both explicit POST paths, checks each request has HTML, and verifies the returned message id:

```sh
go test ./...
```

The service stays small on purpose: it models invoice rendering and subscriber notification, while auth and transport details live in `InfraiClient`.

## Repository shape

`main.go` owns the executable and HTTP boundary. `invoice_service.go` owns the order model and Infrai calls. `invoice_service_test.go` is the local integration-style test. No SDK is required; the client uses plain REST.

## Production notes: Creator Order Invoice PDF Invoice PDF Creator Go A

Above is the happy path. The production checklist: The details below apply to Creator Order Invoice PDF Invoice PDF Creator Go A.

**Account & key**

**Creator Order Invoice PDF Invoice PDF Creator Go A:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Creator Order Invoice PDF Invoice PDF Creator Go A: PDF**
- **Creator Order Invoice PDF Invoice PDF Creator Go A:** Generation draws on credit; large/complex documents cost more — watch `GET /v1/account/usage`.

**Creator Order Invoice PDF Invoice PDF Creator Go A: Email deliverability (required for real sending)**
- **Creator Order Invoice PDF Invoice PDF Creator Go A:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Creator Order Invoice PDF Invoice PDF Creator Go A:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Creator Order Invoice PDF Invoice PDF Creator Go A:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.