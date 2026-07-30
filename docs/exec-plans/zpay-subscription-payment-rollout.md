# ZPAY Subscription Payment Rollout

Related requirement:
`docs/product-specs/zpay-subscription-payment.md`.

## Status

- [x] Create four disabled 30-day subscription plans and their groups.
- [x] Configure recharge limits, timeout, pending-order limit, and zero surcharge.
- [x] Keep global payment, visible methods, and plan sales disabled.
- [x] Confirm the public webhook and return page are reachable over HTTPS.
- [x] Confirm production has no provider instance and no payment orders.
- [x] Validate focused backend and frontend payment tests.
- [x] Submit the ZPAY Alipay small-merchant application and pay the opening fee.
- [x] Change EasyPay order recovery to use ZPAY's documented GET query API.
- [x] Integrate the validated query compatibility fix into local `anonym/custom`.
- [ ] Push `anonym/custom` and deploy the fix after explicit authorization.
- [ ] Wait for merchant approval and complete owner Alipay signing.
- [ ] Store the approved PID and PKey in an encrypted, initially disabled provider instance.
- [ ] Verify a real CNY 10 balance recharge and duplicate callback handling.
- [ ] Verify a real CNY 39 plan purchase and 30-day subscription activation.
- [ ] Enable Alipay visibility, plan sales, and global payment only after verification.
- [ ] Observe revenue, payment fees, complaints, refunds, and account cost for 14 days.

## Safety Boundary

Do not store credentials in Git, shell history, chat output, or this plan. Do not
enable payment or plan sales before both real-payment checks pass. Production
database changes require a fresh backup and a verified rollback path.

## Validation

- Production `payment_enabled=false`.
- Production Alipay and WeChat visible-method switches are false.
- Production provider instance count is zero and payment order count is zero.
- All four subscription plans have `for_sale=false`.
- The webhook GET and POST probes returned HTTP 200 with `success` while no
  provider was configured; the result page returned HTTP 200.
- Focused Go payment packages passed before implementation.
- Five focused frontend payment test files passed (54 tests).
- ZPAY's live documented query endpoint returned JSON for GET and an empty body
  for the equivalent POST request with non-merchant test parameters.
- `go test ./internal/payment/provider -run TestEasyPayQueryOrderStatusMapping -count=1`:
  failed before the implementation and passed afterward.
- `go test ./... -count=1`: passed.
- `go build ./cmd/server`: passed.
- `git diff --check`: passed.
- Harness docs audit: no errors; seven pre-existing warnings remain outside this
  task's documents.
