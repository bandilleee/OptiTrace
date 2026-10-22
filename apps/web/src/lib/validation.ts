export function nameError(value: string): string | undefined {
  const name = value.trim();
  if (!name) return 'Enter a name.';
  if (/[\p{Cc}]/u.test(name)) return 'Use a name without control characters.';
  if (new TextEncoder().encode(name).length > 120)
    return 'This name is too long. Try a shorter name.';
}
export function emailError(value: string): string | undefined {
  const email = value.trim();
  if (
    new TextEncoder().encode(email).length > 254 ||
    !/^[A-Za-z0-9!#$%&'*+\/=?^_`{|}~-]+(?:\.[A-Za-z0-9!#$%&'*+\/=?^_`{|}~-]+)*@[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?)+$/.test(
      email,
    )
  )
    return 'Enter a valid Google account email, such as teammate@example.com.';
  if (
    email.split('@')[0].length > 64 ||
    email
      .split('@')[1]
      .split('.')
      .some((part) => part.length > 63)
  )
    return 'Enter a valid Google account email.';
}
export function integerError(value: string, min: number, max: number): string | undefined {
  if (
    !/^-?\d+$/.test(value) ||
    !Number.isSafeInteger(Number(value)) ||
    Number(value) < min ||
    Number(value) > max
  )
    return `Enter a whole number between ${min} and ${max}.`;
}
