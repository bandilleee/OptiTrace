CREATE OR REPLACE FUNCTION immutable_policy() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF TG_OP='UPDATE' AND (OLD.kind='policy' OR NEW.kind='policy') THEN
  IF NEW.kind IS DISTINCT FROM OLD.kind OR NEW.id IS DISTINCT FROM OLD.id OR NEW.tenant_id IS DISTINCT FROM OLD.tenant_id THEN
   RAISE EXCEPTION 'policy identity immutable';
  END IF;
  IF OLD.data->>'status' IN ('approved','active') THEN
   IF (NEW.data - 'status') IS DISTINCT FROM (OLD.data - 'status') OR
      NOT (NEW.data->>'status'=OLD.data->>'status' OR (OLD.data->>'status'='approved' AND NEW.data->>'status'='active')) OR
      NEW.data->>'status' IS NULL THEN
    RAISE EXCEPTION 'approved policy immutable';
   END IF;
  END IF;
 END IF;
 IF NEW.kind='policy' THEN
  IF NEW.data->>'status' IS NULL OR NEW.data->>'status' NOT IN ('draft','submitted','approved','active') THEN
   RAISE EXCEPTION 'invalid policy status';
  END IF;
  IF NEW.data->>'status' IN ('approved','active') AND
     (coalesce(NEW.data->>'reviewer_id','')='' OR coalesce(NEW.data->>'author_id','')='' OR NEW.data->>'author_id'=NEW.data->>'reviewer_id') THEN
   RAISE EXCEPTION 'independent reviewer required';
  END IF;
 END IF;
 RETURN NEW;
END $$;
DROP TRIGGER policy_immutable ON resources;
CREATE TRIGGER policy_immutable BEFORE INSERT OR UPDATE ON resources FOR EACH ROW EXECUTE FUNCTION immutable_policy();
