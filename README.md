# poly-starcoin-relayer

## Remove Poly(to-Starcoin) Tx. blocking process

Set a `poly_tx` row's status to 'TBR'(To Be Removed):

```sql
update poly_tx set status = 'TBR' where tx_index = {TX_INDEX};
```

The scheduled tasks will check it, if suitable, remove it from `poly_tx` table and backup to `removed_poly_tx` table.

Check `removed_poly_tx` table:

```sql
select id, origin_tx_index, from_chain_id, poly_tx_hash, smt_non_membership_root_hash, status, starcoin_tx_hash from removed_poly_tx order by id desc limit 10;
```

## Push removed Poly(to-Starcoin) Tx. back to process 

Set a `removed_poly_tx` row's status to 'TBP'(To Be Pushed-back):

```sql
update removed_poly_tx set status = 'TBP' where id = {ROW_ID};
```

The scheduled tasks will remove it from `removed_poly_tx` table and push back to `poly_tx` table.

Check `poly_tx` table:

```sql
select tx_index, from_chain_id, poly_tx_hash, smt_non_membership_root_hash, status, retry_count, starcoin_tx_hash from poly_tx order by tx_index desc limit 10;
```

## Is-Accept-Token checking of Starcoin account

Before relay Poly Tx.(to-Starcoin), realyer will check the target(Starcoin) account is-accept the token in cross-chain transfer Tx., if account not accept, save it in `poly_tx_retry` table temporary. The scheduled tasks will check it periodically, and commit it to Starcoin when account accept.  

Check `poly_tx_retry` table:

```sql
select from_chain_id, tx_hash, starcoin_status, check_starcoin_count, check_starcoin_message, fee_status from poly_tx_retry;
```

## Disable (Starcoin)to Poly relaying

Can use command flag `to-poly-disabled` to do this:

```shell
./poly-starcoin-relayer --cliconfig ./config-testnet.json --to-poly-disabled true
```

## Disable (Poly)to Starcoin relaying

Can use command flag `to-starcoin-disabled` to do this:

```shell
./poly-starcoin-relayer --cliconfig ./config-testnet.json --to-starcoin-disabled true
```

