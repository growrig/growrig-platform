import { redirect } from '@sveltejs/kit';

// Knowledge was renamed to Library. Keep old links and bookmarks working.
export function load() {
	throw redirect(307, '/library');
}
