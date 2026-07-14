// Shared error helpers.

/** Extract a human-readable message from a caught value. Thrown `Error`s carry
 *  their message (the REST client surfaces the server's error body this way);
 *  anything else falls back to the caller's default. */
export function errMsg(e: unknown, fallback = 'Something went wrong'): string {
	return e instanceof Error ? e.message : fallback;
}
