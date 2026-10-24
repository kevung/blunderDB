// The backend's hold on the library a write names rows of (Database.HoldLibrary): apart, so that
// the tests, which have no backend, answer it once for all (test-setup.js).
export { HoldLibrary, ReleaseLibrary } from '../../wailsjs/go/database/Database.js';
