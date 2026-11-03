// Product variables in prompt templates. Mirrors products.SlotsNeeded on the
// backend, which is the authority — this only decides whether to ask which
// products to use before generating.
//
//   {{product.name}} {{product.price}} {{product.note}}   the first product
//   {{product2.name}} …                                    the second
// ("product" and "product1" are the same.)

const VARIABLE = /\{\{\s*(product[12]?)\.(name|price|note)\s*\}\}/g;

/** How many products a template needs chosen: 0, 1 or 2. */
export function productSlotsNeeded(body: string): 0 | 1 | 2 {
  let need: 0 | 1 | 2 = 0;
  for (const m of body.matchAll(VARIABLE)) {
    const slot = m[1] === "product2" ? 2 : 1;
    if (slot > need) need = slot;
  }
  return need;
}
