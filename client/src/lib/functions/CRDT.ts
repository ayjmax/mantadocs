import { DECIMAL_BASE, type Identifier } from "$lib/types/CRTD-types";


function comparePosition(p1: Identifier[], p2: Identifier[]): 1 | -1 | 0 {
  for (let i = 0; i < Math.min(p1.length, p2.length); i++) {
    const comp = compareIdentifier(p1[i], p2[i]);
    if (comp !== 0) {
      return comp;
    }
  }
  if (p1.length < p2.length) {
    return - 1;
  } else if (p1.length > p2.length) {
    return 1;
  } else {
    return 0;
  }
}

function compareIdentifier(i1: Identifier, i2: Identifier): 1 | -1 | 0 {
  if (i1.digit < i2.digit) {
    return -1;
  } else if (i1.digit > i2.digit) {
    return 1;
  } else {
    if (i1.siteID < i2.siteID) {
        return -1;
    } else if (i1.siteID > i2.siteID) {
        return 1;
    } else {
        return 0;
    }
  }
}

function createIdentifier(digit: number, siteID: string) {
  if (digit > DECIMAL_BASE) {
    throw new Error(`Digit given is larger than DECIMAL_BASE: ${digit}`);
  }
  let newIdentifier: Identifier = { digit, siteID };
  return newIdentifier
}

function head(position: Identifier[]): Identifier | null {
  return position && position[0] ? position[0] : null;
}

function restOf(postion: Identifier[]): Identifier[] {
  return postion.slice(1);
}

function decimalFromIdentifierList(identifiers: Identifier[]): number[] {
  return identifiers.map(ident => ident.digit);
}

function decimalToIdentifierList(
  nums: number[],
  behind: Identifier[],
  ahead: Identifier[],
  creationSite: string
): Identifier[] {
  return nums.map((digit, index) => {
    if (index === nums.length - 1) {
      return createIdentifier(digit, creationSite);
    } else if (index < behind.length && digit === behind[index].digit) {
      return createIdentifier(digit, behind[index].siteID);
    } else if (index < ahead.length && digit === ahead[index].digit) { // TODO: Why is this part necessary?
      return createIdentifier(digit, ahead[index].siteID);
    } else {
      return createIdentifier(digit, creationSite);
    }
  });
}

/**
 * Perform subtraction w/ carry
 * @param n1 Numbers array
 * @param n2 Numbers array
 * @returns Numbers array representing difference
 */
function subtractGreaterThan(n1: number[], n2: number[]): number[] {
  let carry = 0;
  let diff: number[] = Array(Math.max(n1.length, n2.length));
  for (let i = diff.length - 1; i >= 0; i--) {
    const d1 = (n1[i] || 0) - carry;
    const d2 = (n2[i] || 0);
    if (d1 < d2) {
      carry = 1;
      diff[i] = d1 + DECIMAL_BASE - d2;
    } else {
      carry = 0;
      diff[i] = d1 - d2;
    }
  }
  return diff;
}

function add(n1: number[], increment: number[]): number[] {
  let carry = 0;
  let sum: number[] = Array(Math.max(n1.length, increment.length));
  for (let i = sum.length - 1; i >= 0; i--) {
    const partialSum = (n1[i] || 0) + (increment[i] || 0) + carry;
    sum[i] = partialSum % DECIMAL_BASE;
    carry = partialSum > DECIMAL_BASE ? 1 : 0;
  }
  if (carry > 0) {
    throw new Error("Sum is greater than one, cannot be represented by this type.");
  }
  return sum;
}

function increment(n1: number[], delta: number[]): number[] {
  const firstNonzeroDigit = delta.findIndex(x => x !== 0);
  const increment = delta.slice(0, firstNonzeroDigit).concat([0, 1]);
  const v1 = add(n1, increment);
  const v2 = v1[v1.length - 1] === 0 ? add(v1, increment) : v1;
  return v2;
}

function generatePositionBetween(position1: Identifier[], position2: Identifier[], siteID: string): Identifier[] {
  // Get either the head of the position, or fallback to default value
  const head1 = head(position1) || createIdentifier(0, siteID);
  const head2 = head(position2) || createIdentifier(DECIMAL_BASE, siteID);

  if (head1.digit !== head2.digit) {
    // Case 1: Head digits are different
    const n1 = decimalFromIdentifierList(position1);
    const n2 = decimalFromIdentifierList(position2);
    const delta = subtractGreaterThan(n2, n1);
    // Increment n1 by some amount less than delta
    const newPosNums = increment(n1, delta);
    return decimalToIdentifierList(newPosNums, position1, position2, siteID);
  } else {
    if (head1.siteID.localeCompare(head2.siteID) < 0) {
      // Case 2: Head digits are the same, sites are different
      return [head1].concat(generatePositionBetween(restOf(position1), [], siteID));
    } else if (head1.siteID.localeCompare(head2.siteID) === 0) {
      // Case 3: Head digits and sites are the same
      return [head1].concat(generatePositionBetween(restOf(position1), restOf(position2), siteID));
    } else {
      throw new Error("Invalid siteID ordering or positions");
    }
  }
}


export { compareIdentifier, comparePosition, generatePositionBetween };