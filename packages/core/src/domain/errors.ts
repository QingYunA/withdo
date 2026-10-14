/** A caller-fixable problem. `field` names the offending input so tool errors can point at it. */
export class DomainError extends Error {
  readonly field: string | undefined;

  constructor(message: string, field?: string) {
    super(field ? `${field}: ${message}` : message);
    this.name = "DomainError";
    this.field = field;
  }
}
