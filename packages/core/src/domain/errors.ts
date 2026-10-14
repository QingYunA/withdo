/** A caller-fixable problem. `field` names the offending input so tool errors can point at it. */
export class DomainError extends Error {
  readonly field: string | undefined;

  constructor(message: string, field?: string) {
    super(field ? `${field}: ${message}` : message);
    this.name = "DomainError";
    this.field = field;
  }
}

/** A referenced row does not exist. Still a DomainError, so callers that catch the base class are unaffected. */
export class NotFoundError extends DomainError {
  constructor(message: string, field?: string) {
    super(message, field);
    this.name = "NotFoundError";
  }
}
