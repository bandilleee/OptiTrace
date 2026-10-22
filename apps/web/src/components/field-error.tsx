export function FieldError({ id, error }: { id: string; error?: string }) {
  return error ? (
    <small id={id} className="field-error" role="alert">
      {error}
    </small>
  ) : null;
}
