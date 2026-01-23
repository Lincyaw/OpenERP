/**
 * Create a store with selectors
 *
 * This utility helps create stores with automatic selector generation
 * for better performance (prevents unnecessary re-renders)
 *
 * @example
 * ```tsx
 * const useCounterStore = createStoreWithSelectors<CounterState & CounterActions>(
 *   (set) => ({
 *     count: 0,
 *     increment: () => set((state) => ({ count: state.count + 1 })),
 *   })
 * )
 *
 * // Use with auto-generated selectors
 * const count = useCounterStore.use.count()
 * const increment = useCounterStore.use.increment()
 * ```
 */

import { create, type StateCreator, type StoreApi, type UseBoundStore } from 'zustand'

type WithSelectors<S> = S extends { getState: () => infer T }
  ? S & { use: { [K in keyof T]: () => T[K] } }
  : never

/**
 * Add automatic selectors to a store
 *
 * This enables accessing individual state slices without subscribing
 * to the entire store, improving performance.
 */
export function createSelectors<S extends UseBoundStore<StoreApi<object>>>(store: S) {
  const storeWithSelectors = store as WithSelectors<typeof store>
  storeWithSelectors.use = {} as Record<string, () => unknown>

  for (const key of Object.keys(store.getState())) {
    ;(storeWithSelectors.use as Record<string, () => unknown>)[key] = () =>
      store((state) => (state as Record<string, unknown>)[key])
  }

  return storeWithSelectors
}

/**
 * Create a store with selectors in one step
 */
export function createStoreWithSelectors<T extends object>(
  initializer: StateCreator<T, [], []>
): WithSelectors<UseBoundStore<StoreApi<T>>> {
  return createSelectors(create<T>()(initializer))
}
