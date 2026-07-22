/**
 * Minimal ambient declarations for the native WeChat mini program runtime.
 *
 * We intentionally type the framework surface (wx / Page / Component / App ...)
 * as `any` so that `tsc --checkJs` validates our own JS (syntax, undefined
 * references, wrong local call signatures) without requiring the full
 * `miniprogram-api-typings` package or fighting the loosely typed `this`
 * inside Page/Component method callbacks.
 */

declare const wx: any;
declare function App(options: any): void;
declare function Page(options: any): void;
declare function Component(options: any): void;
declare function Behavior(options: any): any;
declare function getApp(): any;
declare function getCurrentPages(): any[];
declare function requirePlugin(name: string): any;
declare function getRegExp(pattern: string, flags?: string): RegExp;

declare const __wxConfig: any;

// CommonJS module globals used across the project.
declare const module: { exports: any };
declare const require: (path: string) => any;
declare const global: any;

// Host timer globals (not part of the ES2019 lib we compile against).
declare function setTimeout(handler: (...args: any[]) => void, timeout?: number, ...args: any[]): number;
declare function clearTimeout(handle?: number): void;
declare function setInterval(handler: (...args: any[]) => void, timeout?: number, ...args: any[]): number;
declare function clearInterval(handle?: number): void;
