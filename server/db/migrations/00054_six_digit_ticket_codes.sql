-- +goose Up
-- Ticket verification codes are intentionally short enough for staff to enter
-- manually. Assign every existing ticket a unique six-digit code; new tickets
-- use cryptographically secure random codes with duplicate-key retries.

CREATE TEMPORARY TABLE tmp_ticket_code_capacity (
  ticket_count BIGINT NOT NULL,
  CONSTRAINT chk_ticket_code_capacity CHECK (ticket_count <= 1000000)
);

INSERT INTO tmp_ticket_code_capacity (ticket_count)
SELECT COUNT(*) FROM tickets;

SET @ticket_code_offset = FLOOR(RAND() * 1000000);
SET @ticket_code_position = 0;

CREATE TEMPORARY TABLE tmp_ticket_six_digit_codes (
  ticket_id BIGINT UNSIGNED NOT NULL,
  verification_code CHAR(6) NOT NULL,
  PRIMARY KEY (ticket_id),
  UNIQUE KEY uq_tmp_ticket_six_digit_code (verification_code)
);

INSERT INTO tmp_ticket_six_digit_codes (ticket_id, verification_code)
SELECT ranked.id,
       LPAD(MOD((@ticket_code_position := @ticket_code_position + 1) * 7919 + @ticket_code_offset, 1000000), 6, '0')
FROM (
  SELECT id
  FROM tickets
  ORDER BY SHA2(CONCAT(id, @ticket_code_offset), 256), id
  LIMIT 18446744073709551615
) ranked;

-- Move old values out of the six-digit namespace before applying the mapping,
-- so deployments containing an earlier manually-created numeric code cannot
-- hit a transient unique-key collision during the update.
UPDATE tickets
SET verification_code = CONCAT('~', LPAD(HEX(id), 16, '0'));

UPDATE tickets t
JOIN tmp_ticket_six_digit_codes mapped ON mapped.ticket_id = t.id
SET t.verification_code = mapped.verification_code;

DROP TEMPORARY TABLE tmp_ticket_six_digit_codes;
DROP TEMPORARY TABLE tmp_ticket_code_capacity;

-- +goose Down
-- Existing random verification codes cannot be reconstructed safely.
SELECT 1;
