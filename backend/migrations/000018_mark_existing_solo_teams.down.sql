UPDATE teams 
SET is_solo = FALSE, is_auto_created = FALSE
WHERE is_solo = TRUE AND is_auto_created = TRUE;
