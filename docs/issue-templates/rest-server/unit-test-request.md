# 🧪 Unit Test Implementation

### Issue Title (Name issue with ticket number, priority level and title)

* [DYS-0928] Implement Unit Tests for `CreateCoupon` method located in `./internal/services/coupon_service.go`
* [DYS-3840] Implement Unit Tests for all the methods in `utils/string_formatter.go`

---

### What code are we testing?

* **File/Package:** <e.g., internal/services/coupon_service.go>
* **Function(s)/Method(s) to test:** <e.g., `GetUserByID`, `CreateUser`, `UpdateUserProfile`>

---

### What are we testing for? (Scope/Coverage)

* *Describe the main scenarios/behaviors this unit test aims to cover.* (e.g., "Verify `CreateUser` successfully creates a user, handles duplicate emails, and returns validation errors correctly.")
    * Examples:
        * Happy path / successful execution
        * Edge cases (empty inputs, zero values)
        * Error conditions (invalid arguments, dependency failures)
        * Specific return values
        * Concurrency safety (if applicable)

---

### Key Test Details & Considerations

* **Mocking Strategy:**
    * <e.g., Using command `make mock` at the root of project to generate the mock implementions>
    * <e.g., No mocking, testing pure functions>
* **Test Data Setup:**
    * <e.g., Inline struct literals>
    * <e.g., Test fixtures defined in a helper function>
* **Assertions Used:** <e.g., `testify/assert`>
* **Dependencies to Mock (if any):**
    * <e.g., `CouponRepository` interface>
    * <e.g., `Redis` interface>
    * <e.g., External API client>

---

### More Context you want to provide?

* *Any specific instructions or existing test patterns to follow?* (e.g., "Refer to the existing tests in `order_service_test.go` for structure." or "Focus on testing the business logic, not database interactions.")

---
