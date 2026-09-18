/**
 * Replaces next/navigation's notFound(). Throwing propagates to the nearest
 * route ErrorBoundary, which renders the 404 view when it sees this type --
 * mirroring how Next unwound to the not-found boundary.
 */
export class NotFoundError extends Error {
  constructor(message = 'Not found') {
    super(message);
    this.name = 'NotFoundError';
  }
}

export function notFound(): never {
  throw new NotFoundError();
}

export function isNotFoundError(error: unknown): error is NotFoundError {
  return error instanceof NotFoundError;
}
