import { getCountrySpecifications } from "ibantools";

// IBAN format token used by react-imask:
//   '0' = digit (0-9)
//   'a' = letter (A-Z, our prepareChar upper-cases it)
//   'X' = custom alphanumeric ([A-Z0-9]) — see `IBAN_DEFINITIONS`
type Token = "0" | "a" | "X";

interface IBANMask {
  startsWith: string;
  mask: string;
  chars: number;
}

// Parse strings like "^[0-9]{16}$" or "^[A-Z]{4}[0-9]{14}$" or
// "^[0-9]{10}[A-Z0-9]{11}[0-9]{2}$" into a per-position token array.
function parseBbanRegexp(bbanRegexp: string): Token[] {
  const body = bbanRegexp.replace(/^\^/, "").replace(/\$$/, "");
  const re = /\[([^\]]+)\]\{(\d+)\}/g;
  const tokens: Token[] = [];
  let m: RegExpExecArray | null;
  while ((m = re.exec(body)) !== null) {
    const charClass = m[1];
    const count = parseInt(m[2], 10);
    let t: Token = "X";
    if (charClass === "0-9") t = "0";
    else if (charClass === "A-Z") t = "a";
    for (let i = 0; i < count; i++) tokens.push(t);
  }
  return tokens;
}

// Build "XX00 GGGG GGGG …" with the BBAN broken into groups of 4.
function buildMask(code: string, chars: number, bbanTokens: Token[]): string {
  let mask = `${code}00`;
  for (let i = 0; i < bbanTokens.length; i += 4) {
    mask += " " + bbanTokens.slice(i, i + 4).join("");
  }
  return mask;
}

// Fallback used until the country code is known (or for unknown countries).
// 2 letters + 2 digits + 30 alphanumeric, grouped in 4s — matches the longest
// theoretical IBAN. `isValidIBAN` is still the final authority.
const FALLBACK_MASK = "aa00 XXXX XXXX XXXX XXXX XXXX XXXX XXXX XX";

function buildAllMasks(): IBANMask[] {
  const specs = getCountrySpecifications();
  const masks: IBANMask[] = [];
  for (const [code, spec] of Object.entries(specs)) {
    if (!spec.chars || !spec.bban_regexp) continue;
    const tokens = parseBbanRegexp(spec.bban_regexp);
    if (tokens.length !== spec.chars - 4) continue;
    masks.push({
      startsWith: code,
      mask: buildMask(code, spec.chars, tokens),
      chars: spec.chars,
    });
  }
  return masks;
}

const COMPILED_MASKS = buildAllMasks();

export const IBAN_DEFINITIONS = {
  X: /[A-Z0-9]/,
};

// Shape consumed by IMaskInput's dynamic mask. The dispatch reads the first
// 2 typed characters and selects the matching country mask, or falls back.
export const IBAN_DYNAMIC_MASK = {
  mask: [...COMPILED_MASKS, { mask: FALLBACK_MASK, startsWith: "" }],
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  dispatch: (appended: string, dynamicMasked: any) => {
    const candidate = (dynamicMasked.value + appended).slice(0, 2).toUpperCase();
    const match = dynamicMasked.compiledMasks.find(
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (m: any) => m.startsWith && m.startsWith === candidate,
    );
    return match ?? dynamicMasked.compiledMasks[dynamicMasked.compiledMasks.length - 1];
  },
};
