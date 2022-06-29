export function match(param: string) {
 return /^[A-Za-z0-9]{10}$/.test(param);
}