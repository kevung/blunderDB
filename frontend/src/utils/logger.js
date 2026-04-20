/* eslint-disable no-console */
const isDev = import.meta.env.DEV;

export const logger = {
    log: (...args) => isDev && console.log(...args),
    warn: (...args) => isDev && console.warn(...args),
    error: (...args) => console.error(...args),
    debug: (...args) => isDev && console.debug(...args)
};
