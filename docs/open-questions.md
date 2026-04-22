# Open Questions

Open points that still need to be clarified before they can be specified as features.

---

## OQ-1: Documents in the Registration Form

**Context:**
In the consent section of the registration form, the member confirms having read the privacy policy. In addition, there are other documents that must be made accessible to a new member before or during registration, for example:

- EEG statutes
- supplier obligations
- privacy policy
- potentially further legal documents

**Open questions:**
- Which documents are specifically required?
- Must all EEGs use the same documents, or are there EEG-specific documents?
- How are the documents provided? (Direct link, upload, static URL, CMS?)
- Must consent to each document be recorded individually?
- Must the timestamp of consent per document be stored?

**Impact on existing implementation:**
The fields `privacy_version` and `privacy_accepted_at` currently cover only the privacy policy. If extended to multiple documents, a dedicated consent model would be required.

**Status:** Unresolved — must be coordinated with the business owner before implementation.

---

## OQ-2: Formal Requirements for the SEPA Direct Debit Mandate

**Context:**
The current implementation captures consent to the SEPA direct debit mandate as a simple checkbox in the registration form. It is unclear whether this meets the formal requirements for a valid SEPA mandate.

**Open questions:**
- Does a digital checkbox consent constitute a legally valid SEPA mandate?
- What mandatory details must a SEPA mandate contain (e.g. creditor ID, mandate reference)?
- Must the mandate be delivered to the member (e.g. by email)?
- Must a mandate reference be assigned and stored per member?
- Are there requirements for the retention of the mandate?

**Impact on existing implementation:**
The fields `sepa_mandate_accepted` and `sepa_mandate_accepted_at` are a minimal skeleton. If formal requirements apply, additional fields (mandate reference, creditor ID, delivery confirmation) as well as a dedicated process step would be necessary.

**Status:** Unresolved — legal and banking review required.
