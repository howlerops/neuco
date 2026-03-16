/**
 * Type compatibility layer.
 *
 * Re-exports all types from the hand-maintained types module.
 * New code should import from this file. Over time, individual types
 * will be replaced with OpenAPI-derived equivalents from v1.d.ts.
 *
 * Run `pnpm generate:api` to regenerate OpenAPI types in v1.d.ts.
 */
export * from './types';

// Also re-export the CamelCaseKeys utility for future OpenAPI migrations.
import type { components } from './v1';

type SnakeToCamel<S extends string> = S extends `${infer Head}_${infer Tail}`
  ? `${Head}${Capitalize<SnakeToCamel<Tail>>}`
  : S;

type Primitive = string | number | boolean | bigint | symbol | null | undefined;

export type CamelCaseKeys<T> = T extends Primitive
  ? T
  : T extends (...args: never[]) => unknown
    ? T
    : T extends readonly (infer U)[]
      ? CamelCaseKeys<U>[]
      : T extends object
        ? { [K in keyof T as K extends string ? SnakeToCamel<K> : K]: CamelCaseKeys<T[K]> }
        : T;

// OpenAPI schema reference for typed client consumers
export type ApiSchemas = components['schemas'];
