UPDATE teams 
SET is_solo = TRUE, is_auto_created = TRUE
WHERE id IN (
    SELECT t.id 
    FROM teams t
    JOIN users u ON u.team_id = t.id
    GROUP BY t.id 
    HAVING COUNT(*) = 1 AND t.name = MAX(u.username)
);
