export function generateRandomString(size) {
  const characters =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  let result = "";
  for (let i = 0; i < size; i++) {
    const randomIndex = Math.floor(Math.random() * characters.length);
    result += characters.charAt(randomIndex);
  }
  return result;
}

export function calculateCharLength(str) {
  if (!str) return 0;
  let totalLength = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charAt(i);
    if (/[\u4e00-\u9fa5\u3000-\u303f\uff00-\uffef]/.test(char)) {
      totalLength += 2;
    } else {
      totalLength += 1;
    }
  }
  return totalLength;
}